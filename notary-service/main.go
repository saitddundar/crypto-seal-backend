package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// ==================== MODELS ====================

// SealRecord - Kayıtlı mühür bilgisi
type SealRecord struct {
	ID        string    `json:"id"`
	Hash      string    `json:"hash"`
	Timestamp time.Time `json:"timestamp"`
	Text      string    `json:"text,omitempty"` // Opsiyonel: orijinal metin
}

// SealRequest - Frontend'den gelen mühür isteği
type SealRequest struct {
	Text string `json:"text"`
}

// SealResponse - Mühür oluşturma cevabı
type SealResponse struct {
	ID        string    `json:"id"`
	Hash      string    `json:"hash"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
}

// VerifyRequest - Doğrulama isteği
type VerifyRequest struct {
	Text string `json:"text"`
}

// VerifyResponse - Doğrulama cevabı
type VerifyResponse struct {
	Valid   bool        `json:"valid"`
	Message string      `json:"message"`
	Record  *SealRecord `json:"record,omitempty"`
}

// ErrorResponse - Hata cevabı
type ErrorResponse struct {
	Error string `json:"error"`
}

// ==================== IN-MEMORY STORE ====================

type Store struct {
	mu      sync.RWMutex
	records map[string]*SealRecord // hash -> record
	counter int
}

var store = &Store{
	records: make(map[string]*SealRecord),
	counter: 0,
}

// ==================== HASHER SERVICE CLIENT ====================

// getHasherServiceURL - Hasher service URL'ini environment'dan alır
func getHasherServiceURL() string {
	url := os.Getenv("HASHER_SERVICE_URL")
	if url == "" {
		url = "http://localhost:8081/hash"
	}
	return url
}

// HashRequest - Hasher service'e gönderilen istek
type HashRequest struct {
	Text string `json:"text"`
}

// HashResponse - Hasher service'den gelen cevap
type HashResponse struct {
	Hash string `json:"hash"`
}

// getHashFromService - Hasher service'den hash alır
func getHashFromService(text string) (string, error) {
	reqBody, _ := json.Marshal(HashRequest{Text: text})

	resp, err := http.Post(getHasherServiceURL(), "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", fmt.Errorf("hasher service unreachable: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("hasher service returned status: %d", resp.StatusCode)
	}

	var hashResp HashResponse
	if err := json.NewDecoder(resp.Body).Decode(&hashResp); err != nil {
		return "", fmt.Errorf("failed to decode hash response: %v", err)
	}

	return hashResp.Hash, nil
}

// ==================== HANDLERS ====================

// setCORSHeaders - CORS header'larını ekler
func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")
}

// sealHandler - Metin alır, hash'ler ve kaydeder
func sealHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Only POST method is allowed"})
		return
	}

	var req SealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid JSON body"})
		return
	}

	if req.Text == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Text field is required"})
		return
	}

	// Hasher service'den hash al
	hash, err := getHashFromService(req.Text)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	// In-memory kaydet
	store.mu.Lock()
	store.counter++
	record := &SealRecord{
		ID:        fmt.Sprintf("SEAL-%06d", store.counter),
		Hash:      hash,
		Timestamp: time.Now().UTC(),
		Text:      req.Text,
	}
	store.records[hash] = record
	store.mu.Unlock()

	log.Printf("📝 New seal created: %s", record.ID)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(SealResponse{
		ID:        record.ID,
		Hash:      record.Hash,
		Timestamp: record.Timestamp,
		Message:   "Document sealed successfully",
	})
}

// verifyHandler - Hash doğrulama yapar
func verifyHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Only POST method is allowed"})
		return
	}

	var req VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid JSON body"})
		return
	}

	if req.Text == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Text field is required"})
		return
	}

	// Hasher service'den hash al
	hash, err := getHashFromService(req.Text)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	// Store'da ara
	store.mu.RLock()
	record, exists := store.records[hash]
	store.mu.RUnlock()

	if exists {
		log.Printf("✅ Verification successful: %s", record.ID)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(VerifyResponse{
			Valid:   true,
			Message: "Document verified! This document was sealed.",
			Record:  record,
		})
	} else {
		log.Printf("❌ Verification failed: hash not found")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(VerifyResponse{
			Valid:   false,
			Message: "Document not found. This document was never sealed or has been modified.",
			Record:  nil,
		})
	}
}

// listHandler - Tüm kayıtlı seal'ları listeler
func listHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Only GET method is allowed"})
		return
	}

	store.mu.RLock()
	records := make([]*SealRecord, 0, len(store.records))
	for _, record := range store.records {
		// Text'i gizle (güvenlik için)
		recordCopy := &SealRecord{
			ID:        record.ID,
			Hash:      record.Hash,
			Timestamp: record.Timestamp,
		}
		records = append(records, recordCopy)
	}
	store.mu.RUnlock()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":   len(records),
		"records": records,
	})
}

// healthHandler - Servis sağlık kontrolü
func healthHandler(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "notary-service",
	})
}

func main() {
	// Routes
	http.HandleFunc("/seal", sealHandler)
	http.HandleFunc("/verify", verifyHandler)
	http.HandleFunc("/list", listHandler)
	http.HandleFunc("/health", healthHandler)

	port := ":8082"
	log.Printf("📜 Notary Service starting on port %s", port)
	log.Printf("📍 Endpoints: POST /seal, POST /verify, GET /list, GET /health")
	log.Printf("🔗 Hasher Service: %s", getHasherServiceURL())

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
