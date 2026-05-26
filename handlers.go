package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed web/index.html
var webFS embed.FS

type Server struct {
	sched    *Scheduler
	notifier *Notifier
}

func NewServer(s *Scheduler, n *Notifier) *Server {
	return &Server{sched: s, notifier: n}
}

func (srv *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	indexBytes, _ := fs.ReadFile(webFS, "web/index.html")
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexBytes)
	})

	mux.HandleFunc("GET /api/state", srv.handleState)
	mux.HandleFunc("POST /api/reload", srv.handleReload)
	mux.HandleFunc("POST /api/test", srv.handleTest)
	mux.HandleFunc("POST /api/fire/{id}", srv.handleFire)

	return mux
}

func (srv *Server) handleState(w http.ResponseWriter, r *http.Request) {
	cfg, entries := srv.sched.Snapshot()
	resp := map[string]any{
		"quotes":            cfg.Quotes,
		"general_schedules": cfg.GeneralSchedules,
		"timezone":          cfg.Defaults.Timezone,
		"entries":           entries,
	}
	writeJSON(w, resp)
}

func (srv *Server) handleReload(w http.ResponseWriter, r *http.Request) {
	if err := srv.sched.Reload(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]string{"status": "reloaded"})
}

func (srv *Server) handleTest(w http.ResponseWriter, r *http.Request) {
	if err := srv.notifier.Notify("Motivation", "Test notification — you're all set.", ""); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "fired"})
}

func (srv *Server) handleFire(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	if err := srv.sched.FireByID(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, map[string]string{"status": "fired", "id": id})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
