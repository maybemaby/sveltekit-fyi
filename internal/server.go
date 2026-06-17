package internal

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strconv"
)

type Server struct {
	srv      *http.Server
	db       *sql.DB
	store    *AppStore
	logger   *slog.Logger
	adminKey string
}

type MarkNSFWRequest struct {
	Domain string `json:"domain"`
}

func NewServer(ctx context.Context, logger *slog.Logger) *Server {
	srv := &http.Server{
		Addr: ":8000",
	}

	db, err := ConnectDBReadOnly(ctx)

	if err != nil {
		panic(err)
	}

	store := NewAppStore(db)

	return &Server{
		srv:      srv,
		db:       db,
		store:    store,
		logger:   logger.WithGroup("server"),
		adminKey: os.Getenv("ADMIN_KEY"),
	}
}

func (s *Server) Port(addr string) {
	s.srv.Addr = addr
}

func (s *Server) mountRoutes() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /scans", func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		order := r.URL.Query().Get("order")

		if order != "" && order != "seen_count" && order != "seen_at" {
			http.Error(w, "Invalid order parameter", http.StatusBadRequest)
			return
		}

		if page == "" {
			page = "1"
		}

		pageInt, err := strconv.Atoi(page)
		if err != nil || pageInt < 1 {
			http.Error(w, "Invalid page parameter", http.StatusBadRequest)
			return
		}

		offset := (pageInt - 1) * 30

		domains, err := s.store.GetTopDomains(r.Context(), order, 30, offset)

		if err != nil {
			s.logger.Error("failed to get top domains", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		err = json.NewEncoder(w).Encode(domains)

		if err != nil {
			s.logger.Error("failed to encode response", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	})

	mux.HandleFunc("GET /stats", func(w http.ResponseWriter, r *http.Request) {
		stats, err := s.store.GetStats(r.Context())

		if err != nil {
			s.logger.Error("failed to get stats", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Cache-Control", "public, max-age=300, stale-while-revalidate=600")

		err = json.NewEncoder(w).Encode(stats)

		if err != nil {
			s.logger.Error("failed to encode response", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	})

	mux.HandleFunc("GET /snapshots", func(w http.ResponseWriter, r *http.Request) {
		snapshots, err := s.store.GetSnapshots(r.Context())
		if err != nil {
			s.logger.Error("failed to get snapshots", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Cache-Control", "public, max-age=300, stale-while-revalidate=600")

		err = json.NewEncoder(w).Encode(snapshots)

		if err != nil {
			s.logger.Error("failed to encode response", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	})

	mux.HandleFunc("POST /mark-nsfw", func(w http.ResponseWriter, r *http.Request) {

		authHeader := r.Header.Get("Authorization")

		if s.adminKey == "" || authHeader != "Bearer "+s.adminKey {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var req MarkNSFWRequest

		err := json.NewDecoder(r.Body).Decode(&req)

		if err != nil {
			s.logger.Error("failed to decode request body", "error", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		err = s.store.MarkScanNSFW(r.Context(), req.Domain)

		if err != nil {
			s.logger.Error("failed to mark scan as nsfw", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})

	s.srv.Handler = mux
}

func (s *Server) Start() error {

	s.mountRoutes()

	return s.srv.ListenAndServe()
}
