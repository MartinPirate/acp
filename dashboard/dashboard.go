// Package dashboard provides a web-based monitoring dashboard for ACP
// payment transactions and method status.
package dashboard

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/paideia-ai/acp/core"
)

//go:embed static
var staticFiles embed.FS

// Transaction records a single payment event.
type Transaction struct {
	ID        string        `json:"id"`
	Method    string        `json:"method"`
	Amount    string        `json:"amount"`
	Currency  core.Currency `json:"currency"`
	Success   bool          `json:"success"`
	Timestamp string        `json:"timestamp"`
}

// MethodInfo describes a registered payment method's status.
type MethodInfo struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// Stats holds aggregate statistics for the dashboard.
type Stats struct {
	TotalTransactions int                          `json:"totalTransactions"`
	TotalVolume       map[core.Currency]string     `json:"totalVolume"`
	ByMethod          map[string]MethodStats       `json:"byMethod"`
	SuccessRate       float64                      `json:"successRate"`
}

// MethodStats holds per-method statistics.
type MethodStats struct {
	Count       int    `json:"count"`
	Volume      string `json:"volume"`
	SuccessRate float64 `json:"successRate"`
}

// TransactionStore is a thread-safe in-memory store for transactions.
type TransactionStore struct {
	mu           sync.RWMutex
	transactions []Transaction
	maxSize      int
}

// NewTransactionStore creates a store that retains up to maxSize transactions.
func NewTransactionStore(maxSize int) *TransactionStore {
	if maxSize <= 0 {
		maxSize = 1000
	}
	return &TransactionStore{
		transactions: make([]Transaction, 0, maxSize),
		maxSize:      maxSize,
	}
}

// Record adds a transaction to the store, evicting the oldest if at capacity.
func (s *TransactionStore) Record(tx Transaction) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.transactions) >= s.maxSize {
		// Drop the oldest entry.
		s.transactions = s.transactions[1:]
	}
	s.transactions = append(s.transactions, tx)
}

// Recent returns the most recent n transactions in reverse chronological order.
func (s *TransactionStore) Recent(n int) []Transaction {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.transactions)
	if n <= 0 || n > total {
		n = total
	}
	result := make([]Transaction, n)
	for i := 0; i < n; i++ {
		result[i] = s.transactions[total-1-i]
	}
	return result
}

// Stats computes aggregate statistics from all stored transactions.
func (s *TransactionStore) Stats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := Stats{
		TotalVolume: make(map[core.Currency]string),
		ByMethod:    make(map[string]MethodStats),
	}

	successCount := 0
	// volume tracking per currency using string addition would need big.Rat;
	// for simplicity we track as float64 in this dashboard context.
	volumeFloat := make(map[core.Currency]float64)
	methodSuccess := make(map[string]int)
	methodCount := make(map[string]int)
	methodVolume := make(map[string]float64)

	for _, tx := range s.transactions {
		stats.TotalTransactions++
		if tx.Success {
			successCount++
		}

		amt, _ := strconv.ParseFloat(tx.Amount, 64)
		volumeFloat[tx.Currency] += amt

		methodCount[tx.Method]++
		methodVolume[tx.Method] += amt
		if tx.Success {
			methodSuccess[tx.Method]++
		}
	}

	if stats.TotalTransactions > 0 {
		stats.SuccessRate = float64(successCount) / float64(stats.TotalTransactions)
	}

	for cur, vol := range volumeFloat {
		stats.TotalVolume[cur] = strconv.FormatFloat(vol, 'f', 2, 64)
	}

	for method, count := range methodCount {
		rate := 0.0
		if count > 0 {
			rate = float64(methodSuccess[method]) / float64(count)
		}
		stats.ByMethod[method] = MethodStats{
			Count:       count,
			Volume:      strconv.FormatFloat(methodVolume[method], 'f', 2, 64),
			SuccessRate: rate,
		}
	}

	return stats
}

// Config holds configuration for the dashboard server.
type Config struct {
	ListenAddr string
}

// Server is the dashboard HTTP server.
type Server struct {
	config  Config
	store   *TransactionStore
	methods []MethodInfo
	mu      sync.RWMutex
	mux     *http.ServeMux
}

// NewServer creates a new dashboard server.
func NewServer(config Config, store *TransactionStore) *Server {
	s := &Server{
		config: config,
		store:  store,
		mux:    http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

// SetMethods updates the list of known payment methods.
func (s *Server) SetMethods(methods []MethodInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.methods = make([]MethodInfo, len(methods))
	copy(s.methods, methods)
}

func (s *Server) registerRoutes() {
	// Serve embedded static files at root.
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("dashboard: failed to create sub filesystem: %v", err)
	}
	s.mux.Handle("GET /", http.FileServer(http.FS(staticFS)))

	s.mux.HandleFunc("GET /api/stats", s.handleStats)
	s.mux.HandleFunc("GET /api/transactions", s.handleTransactions)
	s.mux.HandleFunc("GET /api/methods", s.handleMethods)
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
}

// Handler returns the HTTP handler for the dashboard.
func (s *Server) Handler() http.Handler {
	return s.mux
}

// ListenAndServe starts the dashboard server.
func (s *Server) ListenAndServe() error {
	log.Printf("dashboard: listening on %s", s.config.ListenAddr)
	return http.ListenAndServe(s.config.ListenAddr, s.mux)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	stats := s.store.Stats()
	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) handleTransactions(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
			if n > 1000 {
				n = 1000
			}
			limit = n
		}
	}

	txs := s.store.Recent(limit)
	writeJSON(w, http.StatusOK, map[string]any{
		"transactions": txs,
		"count":        len(txs),
	})
}

func (s *Server) handleMethods(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	methods := make([]MethodInfo, len(s.methods))
	copy(methods, s.methods)
	s.mu.RUnlock()

	writeJSON(w, http.StatusOK, methods)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("dashboard: json encode error: %v", err)
	}
}
