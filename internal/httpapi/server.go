// Package httpapi provides the local workspace and the read/share server.
package httpapi

import (
	"bytes"
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/keynertyc/kipuray/internal/browser"
	"github.com/keynertyc/kipuray/internal/casefile"
	"github.com/keynertyc/kipuray/internal/store"
	webassets "github.com/keynertyc/kipuray/internal/web"
)

type Config struct {
	Local      bool
	OwnerToken string
	Host       string
	Browser    browser.Options
	Context    context.Context
}
type Server struct {
	stopCheckpoint context.CancelFunc
	store          *store.Store
	config         Config
	session        string
	mu             sync.Mutex
	capture        *browser.Capture
	running        bool
	mux            *http.ServeMux
}

func New(s *store.Store, c Config) *Server {
	if c.Context == nil {
		c.Context = context.Background()
	}
	a := &Server{store: s, config: c, session: casefile.NewID() + casefile.NewID(), mux: http.NewServeMux()}
	a.mux.HandleFunc("POST /api/login", a.login)
	a.mux.HandleFunc("GET /api/status", func(w http.ResponseWriter, r *http.Request) {
		a.mu.Lock()
		defer a.mu.Unlock()
		status := "idle"
		if a.capture != nil {
			status = a.capture.Status()
		}
		respond(w, map[string]any{"local": c.Local, "capture": status, "running": a.running})
	})
	a.mux.HandleFunc("GET /api/cases", func(w http.ResponseWriter, r *http.Request) { out, err := s.List(r.Context()); reply(w, out, err) })
	a.mux.HandleFunc("POST /api/import", a.importCase)
	a.mux.HandleFunc("GET /api/cases/{id}", func(w http.ResponseWriter, r *http.Request) {
		out, err := s.Get(r.Context(), r.PathValue("id"))
		reply(w, out, err)
	})
	a.mux.HandleFunc("PUT /api/cases/{id}", a.updateCase)
	a.mux.HandleFunc("DELETE /api/cases/{id}", func(w http.ResponseWriter, r *http.Request) {
		reply(w, map[string]bool{"deleted": true}, s.Delete(r.Context(), r.PathValue("id")))
	})
	a.mux.HandleFunc("GET /api/cases/{id}/download", a.download)
	a.mux.HandleFunc("GET /api/cases/{id}/runs", func(w http.ResponseWriter, r *http.Request) {
		out, err := s.Runs(r.Context(), r.PathValue("id"))
		reply(w, out, err)
	})
	a.mux.HandleFunc("POST /api/cases/{id}/run", a.run)
	a.mux.HandleFunc("POST /api/cases/{id}/share", func(w http.ResponseWriter, r *http.Request) {
		token, err := s.Share(r.Context(), r.PathValue("id"), time.Now().Add(24*time.Hour))
		reply(w, map[string]string{"url": "/shared/" + token, "expiresIn": "24 hours"}, err)
	})
	a.mux.HandleFunc("DELETE /api/cases/{id}/share", func(w http.ResponseWriter, r *http.Request) {
		reply(w, map[string]bool{"revoked": true}, s.Revoke(r.Context(), r.PathValue("id")))
	})
	a.mux.HandleFunc("GET /api/shared/{token}", func(w http.ResponseWriter, r *http.Request) {
		out, err := s.Shared(r.Context(), r.PathValue("token"))
		reply(w, out, err)
	})
	a.mux.HandleFunc("POST /api/capture/start", a.startCapture)
	a.mux.HandleFunc("POST /api/capture/stop", a.stopCapture)
	a.mux.HandleFunc("GET /api/capture", func(w http.ResponseWriter, r *http.Request) {
		a.mu.Lock()
		defer a.mu.Unlock()
		if a.capture == nil {
			fail(w, "No active recording", http.StatusNotFound)
			return
		}
		respond(w, a.capture.Snapshot())
	})
	a.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { respond(w, map[string]string{"status": "ok"}) })
	a.mux.HandleFunc("GET /", a.static)
	checkpointCtx, stop := context.WithCancel(c.Context)
	a.stopCheckpoint = stop
	if c.Local {
		go a.checkpoint(checkpointCtx)
	}
	return a
}
func (a *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self' data:; connect-src 'self'; frame-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'self'; form-action 'self'")
	if strings.HasPrefix(r.URL.Path, "/replay/") {
		w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src data: blob:; font-src data:; connect-src 'self'; frame-src 'self'; object-src 'none'; base-uri 'none'; form-action 'none'; frame-ancestors 'self'")
	}
	if a.config.Host != "" && r.Host != a.config.Host {
		fail(w, "Unexpected host", http.StatusForbidden)
		return
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		if origin := r.Header.Get("Origin"); origin != "" {
			u, err := url.Parse(origin)
			if err != nil || u.Host != r.Host {
				fail(w, "Cross-origin request rejected", http.StatusForbidden)
				return
			}
		}
		if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			fail(w, "Cross-site request rejected", http.StatusForbidden)
			return
		}
	}
	if a.config.Local && r.Method == "GET" && !strings.HasPrefix(r.URL.Path, "/api/") {
		http.SetCookie(w, &http.Cookie{Name: "kipuray_session", Value: a.session, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode})
	}
	public := r.URL.Path == "/api/login" || strings.HasPrefix(r.URL.Path, "/api/shared/")
	if strings.HasPrefix(r.URL.Path, "/api/") && !public && !a.authorized(r) {
		fail(w, "Sign in to this workspace", http.StatusUnauthorized)
		return
	}
	a.mux.ServeHTTP(w, r)
}
func (a *Server) authorized(r *http.Request) bool {
	cookie, err := r.Cookie("kipuray_session")
	return err == nil && subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(a.session)) == 1
}
func (a *Server) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token string `json:"token"`
	}
	if !decode(w, r, &in) {
		return
	}
	if a.config.OwnerToken == "" || subtle.ConstantTimeCompare([]byte(in.Token), []byte(a.config.OwnerToken)) != 1 {
		fail(w, "Invalid workspace token", http.StatusUnauthorized)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "kipuray_session", Value: a.session, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"})
	respond(w, map[string]bool{"ok": true})
}
func (a *Server) static(w http.ResponseWriter, r *http.Request) {
	assets := webassets.Assets()
	name := strings.TrimPrefix(r.URL.Path, "/")
	if name == "" {
		name = "index.html"
	}
	b, err := fs.ReadFile(assets, name)
	if err != nil {
		if strings.Contains(name, ".") {
			http.NotFound(w, r)
			return
		}
		b, err = fs.ReadFile(assets, "index.html")
		name = "index.html"
	}
	if err != nil {
		fail(w, "Build the web interface with make build", http.StatusServiceUnavailable)
		return
	}
	http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(b))
}
func (a *Server) importCase(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, casefile.MaxBytes)
	c, err := casefile.Read(r.Body)
	if err != nil {
		fail(w, err.Error(), 400)
		return
	}
	c.ID = casefile.NewID()
	c.CreatedAt = time.Now().UTC()
	err = a.store.Put(r.Context(), c)
	reply(w, c, err)
}
func (a *Server) updateCase(w http.ResponseWriter, r *http.Request) {
	var c casefile.Case
	if !decode(w, r, &c) {
		return
	}
	if c.ID != r.PathValue("id") {
		fail(w, "Case ID mismatch", 400)
		return
	}
	if _, err := a.store.Get(r.Context(), c.ID); err != nil {
		reply(w, nil, err)
		return
	}
	reply(w, &c, a.store.Put(r.Context(), &c))
}
func (a *Server) download(w http.ResponseWriter, r *http.Request) {
	c, err := a.store.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		reply(w, nil, err)
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.kpr"`, c.ID))
	if err := casefile.Write(w, c); err != nil {
		return
	}
}
func (a *Server) startCapture(w http.ResponseWriter, r *http.Request) {
	if !a.config.Local {
		fail(w, "Record on your local workstation", 403)
		return
	}
	var in struct {
		URL         string `json:"url"`
		CaptureJSON bool   `json:"captureJSON"`
	}
	if !decode(w, r, &in) {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.capture != nil || a.running {
		fail(w, "Finish the current recording or run first", 409)
		return
	}
	capture, err := browser.StartCapture(a.config.Context, in.URL, browser.CaptureOptions{Options: a.config.Browser, CaptureJSON: in.CaptureJSON})
	if err != nil {
		fail(w, err.Error(), 400)
		return
	}
	a.capture = capture
	respond(w, capture.Snapshot())
}
func (a *Server) stopCapture(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.capture == nil {
		fail(w, "No recording to stop", 409)
		return
	}
	c := a.capture.Stop()
	a.capture = nil
	c.Title = "Recording · " + time.Now().Format("Jan 02, 15:04")
	reply(w, c, a.store.Put(r.Context(), c))
}
func (a *Server) run(w http.ResponseWriter, r *http.Request) {
	if !a.config.Local {
		fail(w, "Run cases locally or in your own CI", 403)
		return
	}
	var in struct {
		Target    string `json:"target"`
		Before    string `json:"before"`
		VariantID string `json:"variantId"`
	}
	if !decode(w, r, &in) {
		return
	}
	c, err := a.store.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		reply(w, nil, err)
		return
	}
	a.mu.Lock()
	if a.running || a.capture != nil {
		a.mu.Unlock()
		fail(w, "Finish the current browser operation first", 409)
		return
	}
	a.running = true
	a.mu.Unlock()
	defer func() { a.mu.Lock(); a.running = false; a.mu.Unlock() }()
	opts := a.config.Browser
	opts.Headless = true
	var out any
	if in.Before != "" {
		before := browser.Run(r.Context(), c, in.Before, in.VariantID, opts)
		after := browser.Run(r.Context(), c, in.Target, in.VariantID, opts)
		out = browser.Compare(before, after)
	} else {
		out = browser.Run(r.Context(), c, in.Target, in.VariantID, opts)
	}
	reply(w, out, a.store.SaveRun(r.Context(), c.ID, out))
}
func (a *Server) Close() {
	a.stopCheckpoint()
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.capture != nil {
		c := a.capture.Stop()
		if err := a.store.Put(context.Background(), c); err != nil {
			slog.Error("save recording on shutdown", "error", err)
		}
		a.capture = nil
	}
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, casefile.MaxBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		fail(w, "Invalid request: "+err.Error(), 400)
		return false
	}
	if err := dec.Decode(new(any)); err != io.EOF {
		fail(w, "Expected one JSON value", 400)
		return false
	}
	return true
}
func respond(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
func reply(w http.ResponseWriter, v any, err error) {
	if err != nil {
		status := 400
		if errors.Is(err, sql.ErrNoRows) {
			status = 404
		}
		fail(w, err.Error(), status)
		return
	}
	respond(w, v)
}

// checkpoint keeps interrupted recordings inspectable after a process crash.
func (a *Server) checkpoint(ctx context.Context) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.mu.Lock()
			if a.capture != nil {
				c := a.capture.Snapshot()
				c.Title = "Recording in progress · " + c.CreatedAt.Format("Jan 02, 15:04")
				if err := a.store.Put(ctx, c); err != nil {
					slog.Error("checkpoint recording", "error", err)
				}
			}
			a.mu.Unlock()
		}
	}
}
