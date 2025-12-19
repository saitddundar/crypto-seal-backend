package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
)

// HashRequest - Frontend'den gelen istek yapısı
type HashRequest struct {
	Text string `json:"text"`
}

// HashResponse - Frontend'e gönderilecek cevap yapısı
type HashResponse struct {
	Hash string `json:"hash"`
}

// ErrorResponse - Hata durumunda gönderilecek cevap
type ErrorResponse struct {
	Error string `json:"error"`
}

// hashHandler - SHA256 hash işlemini yapar
func hashHandler(w http.ResponseWriter, r *http.Request) {
	// CORS Headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	// Preflight request için
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Sadece POST kabul et
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Only POST method is allowed"})
		return
	}

	// Request body'yi parse et
	var req HashRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid JSON body"})
		return
	}

	// Boş text kontrolü
	if req.Text == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Text field is required"})
		return
	}

	// SHA256 hash hesapla
	hash := sha256.Sum256([]byte(req.Text))
	hashHex := hex.EncodeToString(hash[:])

	// Response gönder
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(HashResponse{Hash: hashHex})
}

// healthHandler - Servis sağlık kontrolü
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy", "service": "hasher-service"})
}

func main() {
	// Routes
	http.HandleFunc("/hash", hashHandler)
	http.HandleFunc("/health", healthHandler)

	port := ":8081"
	log.Printf("🔐 Hasher Service starting on port %s", port)
	log.Printf("📍 Endpoints: POST /hash, GET /health")

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
