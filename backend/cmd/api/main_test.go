package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/thisdev-davi/comunica-faesa/backend/internal/auth"
	"github.com/thisdev-davi/comunica-faesa/backend/internal/events"
	"github.com/thisdev-davi/comunica-faesa/backend/internal/ideas"
)

// Sem cookie, os handlers respondem 401 antes do banco, então rodam sem conexão (nil).
// 401 aqui quer dizer "passou pela proteção e chegou no handler".
func TestRoutesBlockCrossOriginWrites(t *testing.T) {
	a := auth.NewHandler(nil, "id", "secret", "http://localhost/callback", "guild", nil)
	api := routes(a, ideas.NewHandler(nil, a.Viewer), events.NewHandler(nil, a.Viewer))
	cases := []struct {
		name          string
		method, path  string
		secFetchSite  string
		wantForbidden bool
	}{
		{"post de outro *.sslip.io", http.MethodPost, "/api/ideas", "same-site", true},
		{"post de outro site", http.MethodPost, "/api/ideas", "cross-site", true},
		{"logout forçado de outro site", http.MethodPost, "/api/auth/logout", "same-site", true},
		{"apagar ideia de outro site", http.MethodDelete, "/api/ideas/1", "cross-site", true},
		{"criar evento de outro site", http.MethodPost, "/api/events", "cross-site", true},
		{"confirmar presença de outro site", http.MethodPut, "/api/events/1/attendance", "same-site", true},
		{"apagar evento de outro site", http.MethodDelete, "/api/events/1", "cross-site", true},
		{"post da própria página", http.MethodPost, "/api/ideas", "same-origin", false},
		{"post sem navegador (curl)", http.MethodPost, "/api/ideas", "", false},
		{"get de outro site passa", http.MethodGet, "/api/ideas/1", "cross-site", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.secFetchSite != "" {
				req.Header.Set("Sec-Fetch-Site", tc.secFetchSite)
			}
			rec := httptest.NewRecorder()
			api.ServeHTTP(rec, req)
			if got := rec.Code == http.StatusForbidden; got != tc.wantForbidden {
				t.Errorf("status = %d, quer 403: %v", rec.Code, tc.wantForbidden)
			}
			if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
				t.Errorf("X-Content-Type-Options = %q, quer nosniff", got)
			}
			if tc.wantForbidden && rec.Body.String() != `{"error":"cross_origin"}`+"\n" {
				t.Errorf("corpo = %q, quer o erro cross_origin do contrato", rec.Body)
			}
		})
	}
}
