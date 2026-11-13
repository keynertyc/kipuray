package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/keynertyc/kipuray/internal/casefile"
	"github.com/keynertyc/kipuray/internal/store"
)

func TestAuthenticationAndOriginBoundary(t *testing.T) {
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	a := New(s, Config{OwnerToken: strings.Repeat("t", 32), Host: "workspace.test"})
	request := func(method, path, body, origin string, cookies []*http.Cookie) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "http://workspace.test"+path, strings.NewReader(body))
		r.Header.Set("Origin", origin)
		for _, c := range cookies {
			r.AddCookie(c)
		}
		w := httptest.NewRecorder()
		a.ServeHTTP(w, r)
		return w
	}
	if got := request("GET", "/api/cases", "", "", nil).Code; got != 401 {
		t.Fatal(got)
	}
	auth := request("POST", "/api/login", `{"token":"`+strings.Repeat("t", 32)+`"}`, "http://workspace.test", nil)
	if auth.Code != 200 {
		t.Fatal(auth.Body.String())
	}
	cookies := auth.Result().Cookies()
	if got := request("GET", "/api/cases", "", "", cookies).Code; got != 200 {
		t.Fatal(got)
	}
	if got := request("POST", "/api/capture/start", `{"url":"http://localhost:3000"}`, "http://evil.test", cookies).Code; got != 403 {
		t.Fatal(got)
	}
	if got := request("POST", "/api/capture/start", `{"url":"http://localhost:3000"}`, "http://workspace.test", cookies).Code; got != 403 {
		t.Fatal("remote server allowed capture", got)
	}
	wrong := httptest.NewRequest("GET", "http://evil.test/api/cases", nil)
	w := httptest.NewRecorder()
	a.ServeHTTP(w, wrong)
	if w.Code != 403 {
		t.Fatal("host boundary", w.Code)
	}
}
func TestImportExportAndSharing(t *testing.T) {
	ctx := context.Background()
	s, _ := store.Open(t.TempDir())
	defer s.Close()
	a := New(s, Config{Local: true})
	cookies := []*http.Cookie{{Name: "kipuray_session", Value: a.session}}
	do := func(method, path string, body *bytes.Buffer) *httptest.ResponseRecorder {
		var reader *bytes.Reader
		if body == nil {
			reader = bytes.NewReader(nil)
		} else {
			reader = bytes.NewReader(body.Bytes())
		}
		r := httptest.NewRequest(method, "http://localhost"+path, reader)
		for _, c := range cookies {
			r.AddCookie(c)
		}
		w := httptest.NewRecorder()
		a.ServeHTTP(w, r)
		return w
	}
	c := casefile.New("http://localhost:3000")
	var archive bytes.Buffer
	if err := casefile.Write(&archive, c); err != nil {
		t.Fatal(err)
	}
	w := do("POST", "/api/import", &archive)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var imported casefile.Case
	json.Unmarshal(w.Body.Bytes(), &imported)
	if imported.ID == c.ID {
		t.Fatal("import did not assign new identity")
	}
	downloaded := do("GET", "/api/cases/"+imported.ID+"/download", nil)
	if _, err := casefile.Read(downloaded.Body); err != nil {
		t.Fatal(err)
	}
	token, err := s.Share(ctx, imported.ID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	cookies = nil
	if got := do("GET", "/api/shared/"+token, nil); got.Code != 200 {
		t.Fatal(got.Body.String())
	}
	s.Revoke(ctx, imported.ID)
	if got := do("GET", "/api/shared/"+token, nil).Code; got != 404 {
		t.Fatal(got)
	}
}
