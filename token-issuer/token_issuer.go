
/*
==============================
PHASE 5: SESSION TOKEN ISSUER
==============================

This Go service creates secure, expirable tokens for authenticated players.
Tokens are used by the session broker to validate access and forward the user.

------------------------------
🛠 How to Run This Issuer
------------------------------

1. Install Go:
   sudo apt install golang

2. Save this file as: token_issuer.go

3. Build:
   go build -o token-issuer token_issuer.go

4. Run:
   ./token-issuer

------------------------------
📌 What It Does
------------------------------
- Listens on port 4200
- Issues JWT-style tokens on valid auth request (mocked here)
- Tokens include Account ID, Realm ID, expiration
*/

package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "time"

    "github.com/golang-jwt/jwt/v5"
)

var jwtKey = []byte("supersecretkey") // replace with env var or secure config

type Claims struct {
    AccountID string `json:"account_id"`
    RealmID   string `json:"realm_id"`
    jwt.RegisteredClaims
}

type TokenResponse struct {
    Token string `json:"token"`
}

func issueToken(w http.ResponseWriter, r *http.Request) {
    accountID := r.URL.Query().Get("account")
    realmID := r.URL.Query().Get("realm")

    if accountID == "" || realmID == "" {
        http.Error(w, "Missing account or realm", http.StatusBadRequest)
        return
    }

    expiration := time.Now().Add(10 * time.Minute)
    claims := &Claims{
        AccountID: accountID,
        RealmID:   realmID,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(expiration),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenStr, err := token.SignedString(jwtKey)
    if err != nil {
        http.Error(w, "Token signing failed", http.StatusInternalServerError)
        return
    }

    response := TokenResponse{Token: tokenStr}
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func main() {
    http.HandleFunc("/issue", issueToken)

    fmt.Println("Token issuer running on port 4200")
    log.Fatal(http.ListenAndServe(":4200", nil))
}
