
/*
==============================
PHASE 2: SESSION BROKER
==============================

This Go service handles post-authentication user routing. It receives a token (e.g. from login queue),
validates it (placeholder logic for now), and dispatches the player to the appropriate worldserver.

------------------------------
🛠 How to Run This Broker
------------------------------

1. Install Go:
   sudo apt install golang

2. Save this file as: broker.go

3. Build the binary:
   go build -o session-broker broker.go

4. Run the broker:
   ./session-broker

------------------------------
📌 What It Does
------------------------------
- Listens on port 4000 for session token requests
- Validates token (mocked in this version)
- Returns connection target (worldserver IP/port) as JSON
*/

package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "strings"
    "time"
)

type DispatchResponse struct {
    WorldServer string `json:"worldserver"`
    Port        int    `json:"port"`
    Message     string `json:"message"`
}

func validateToken(token string) bool {
    // Placeholder for real JWT validation
    return strings.HasPrefix(token, "valid-")
}

func handleDispatch(w http.ResponseWriter, r *http.Request) {
    token := r.URL.Query().Get("token")
    if token == "" {
        http.Error(w, "Missing token", http.StatusBadRequest)
        return
    }

    if !validateToken(token) {
        http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
        return
    }

    // Simulate realm selection based on load (hardcoded for now)
    response := DispatchResponse{
        WorldServer: "127.0.0.1",
        Port:        8130, // Example worldserver port
        Message:     "Proceed to world server",
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func main() {
    http.HandleFunc("/dispatch", handleDispatch)
    server := &http.Server{
        Addr:         ":4000",
        ReadTimeout:  5 * time.Second,
        WriteTimeout: 5 * time.Second,
    }

    fmt.Println("Session broker listening on port 4000")
    log.Fatal(server.ListenAndServe())
}
