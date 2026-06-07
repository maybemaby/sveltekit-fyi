package internal

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"strings"
)

type Server struct {
	srv *http.Server
	db  *sql.DB
}

func NewServer(ctx context.Context) *Server {
	srv := &http.Server{
		Addr: ":8000",
	}

	db, err := ConnectDBReadOnly(ctx)

	if err != nil {
		panic(err)
	}

	return &Server{
		srv: srv,
		db:  db,
	}
}

func (s *Server) Port(addr string) {
	s.srv.Addr = addr
}

func (s *Server) mountRoutes() {
	mux := http.NewServeMux()

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
