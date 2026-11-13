// Package demo serves a synthetic checkout with deliberately broken and fixed
// versions. The API payload is identical; only the frontend behavior changes.
package demo

import (
	_ "embed"
	"net/http"
	"strings"
)

//go:embed checkout.html
var checkout string

func Handler(fixed bool) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		version := "false"
		if fixed {
			version = "true"
		}
		_, _ = w.Write([]byte(strings.ReplaceAll(checkout, "__FIXED__", version)))
	})
	mux.HandleFunc("GET /api/coupon", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"discount":null,"message":"Coupon not valid"}`))
	})
	return mux
}
