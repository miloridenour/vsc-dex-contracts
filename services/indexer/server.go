package indexer

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

// Server provides HTTP API for indexer read models
type Server struct {
	indexer *Service
	http    *http.Server
}

// NewServer creates a new HTTP server for the indexer
func NewServer(svc *Service, port string) *Server {
	s := &Server{
		indexer: svc,
	}

	r := mux.NewRouter()

	// Pool endpoints
	r.HandleFunc("/api/v1/pools", s.handleGetPools).Methods("GET")
	r.HandleFunc("/api/v1/pools/{id}", s.handleGetPool).Methods("GET")

	// Token endpoints


	// Health check
	r.HandleFunc("/health", s.handleHealth).Methods("GET")

	s.http = &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	return s
}

// Start starts the HTTP server
func (s *Server) Start() error {
	return s.http.ListenAndServe()
}

// Stop stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

// handleGetPools returns all pools
func (s *Server) handleGetPools(w http.ResponseWriter, r *http.Request) {
	pools, err := s.indexer.QueryPools()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pools)
}

// handleGetPool returns a specific pool
func (s *Server) handleGetPool(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	poolID := vars["id"]

	// Get the first read model that supports pool queries
	for _, reader := range s.indexer.readers {
		if dexReader, ok := reader.(*DexReadModel); ok {
			if pool, exists := dexReader.GetPool(poolID); exists {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(pool)
				return
			}
		}
	}

	http.Error(w, "Pool not found", http.StatusNotFound)
}




// handleHealth provides health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"service": "dex-indexer",
	})
}
