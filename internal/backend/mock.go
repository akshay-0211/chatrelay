package backend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type ChatRequest struct {
	UserID string `json:"user_id"`
	Query  string `json:"query"`
}

func StartMockServer() {
	http.HandleFunc("/v1/chat/stream", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var req ChatRequest
		json.NewDecoder(r.Body).Decode(&req)

		time.Sleep(500 * time.Millisecond) // simulate processing
		resp := map[string]string{
			"full_response": "Goroutines are lightweight, concurrent execution units in Go.",
		}
		json.NewEncoder(w).Encode(resp)
	})

	fmt.Println("Mock chat backend listening on :8080")
	http.ListenAndServe(":8080", nil)
}
