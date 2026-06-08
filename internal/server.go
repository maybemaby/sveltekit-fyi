package internal

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

type Server struct {
	srv    *http.Server
	db     *sql.DB
	store  *AppStore
	logger *slog.Logger
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
		srv:    srv,
		db:     db,
		store:  store,
		logger: logger.WithGroup("server"),
	}
}

func (s *Server) Port(addr string) {
	s.srv.Addr = addr
}

func (s *Server) mountRoutes() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /scans", func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		if page == "" {
			page = "1"
		}

		pageInt, err := strconv.Atoi(page)
		if err != nil || pageInt < 1 {
			http.Error(w, "Invalid page parameter", http.StatusBadRequest)
			return
		}

		offset := (pageInt - 1) * 20

		domains, err := s.store.GetTopDomains(r.Context(), 20, offset)

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

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		rows, err := s.db.QueryContext(r.Context(), `SELECT * FROM domains ORDER BY seen_count DESC LIMIT 100`)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// Process rows and write response

		var response strings.Builder

		for rows.Next() {
			var domain string
			var firstSeenAt string
			var lastSeenAt string
			var seenCount int

			err := rows.Scan(&domain, &firstSeenAt, &lastSeenAt, &seenCount)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			_, err = response.WriteString(domain + ":" + strconv.Itoa(seenCount) + "\n")

			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}

		if err := rows.Err(); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Write([]byte(response.String()))
	})

	s.srv.Handler = mux
}

func (s *Server) Start() error {

	s.mountRoutes()

	return s.srv.ListenAndServe()
}
