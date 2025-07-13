package main

/*import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Response struct {
	Status  string    `json:"status"`
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
}

type RateLimitRequest struct {
	ClientID string `json:"client_id"`
	Endpoint string `json:"endpoint"`
	Method   string `json:"method"`
}

func main() {
	fmt.Println("Rate-Limiter Service Starting...")

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api/v1/rate-limit", rateLimitHandler)

	port := ":8080"

	log.Printf("Rate-Limiter startup on %s", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

func healthHandler(wtr http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		http.Error(wtr, "Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	response := Response{
		Status:  "ok",
		Message: "Service is running",
		Time:    time.Now(),
	}

	wtr.Header().Set("Content-Type", "application/json")
	wtr.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(wtr).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(wtr, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func rateLimitHandler(wtr http.ResponseWriter, r *http.Request){
	if (req.method != http.MethodPost) {
		http.Error(wtr, "Method not allowed")
		return
	}

	var rateLimitRequest RateLimitRequest
	err := json.NewDecoder(req.Body).Decode(&rateLimitRequest)
	if (err != nil) {
		log.Printf("Error decoding req body: %v", err)
		http.Error(wtr, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ClientID == "" {
		http.Error(wtr, "ClientID is required", http.StatusBadRequest)
		return
	}

	// TODO - the rate limiting magic happens here
	response := Response{
		Status: "allowed",
		Message: fmt.Sprintf("Req allowed for ClientID %s", rateLimitRequest.ClientID)
		Time: time.Now()
	}

	wtr.Header().Set("Content-type", "application/json")
	wtr.WriteHeader(http.StatusOK)

	err := json.NewEncoder(wtr).Encode(response)
	if err != nil{
		log.Printf("Error encoding response: %v", err)
		Http.Error(wtr, "Internal server error", http.StatusInternalServerError)
	}
}*/
