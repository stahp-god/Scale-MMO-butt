
/*
==============================
PHASE 3: REALM REGISTRY & DISPATCH ENGINE
==============================

This Go service registers active worldservers and provides load-based assignment
to help the session broker send players to the least-loaded realm.

------------------------------
🛠 How to Run This Registry
------------------------------

1. Install Go:
   sudo apt install golang

2. Install Redis:
   sudo apt install redis
   sudo systemctl start redis

3. Save this file as: realm_registry.go

4. Build the binary:
   go build -o realm-registry realm_registry.go

5. Run the registry:
   ./realm-registry

------------------------------
📌 What It Does
------------------------------
- Listens for worldserver pings to register themselves
- Tracks worldserver load in Redis
- Exposes an endpoint for the session broker to fetch the least-loaded realm
*/

package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "sort"
    "strconv"
    "time"

    "github.com/go-redis/redis/v8"
    "golang.org/x/net/context"
)

var (
    ctx = context.Background()
    rdb *redis.Client
)

type WorldServer struct {
    IP       string `json:"ip"`
    Port     int    `json:"port"`
    Players  int    `json:"players"`
    RealmID  string `json:"id"`
    LastSeen int64  `json:"last_seen"`
}

func initRedis() {
    rdb = redis.NewClient(&redis.Options{
        Addr:     "localhost:6379",
        Password: "",
        DB:       0,
    })

    if _, err := rdb.Ping(ctx).Result(); err != nil {
        log.Fatalf("Failed to connect to Redis: %v", err)
    }
}

func registerWorldServer(w http.ResponseWriter, r *http.Request) {
    id := r.URL.Query().Get("id")
    ip := r.URL.Query().Get("ip")
    port := r.URL.Query().Get("port")
    players := r.URL.Query().Get("players")

    if id == "" || ip == "" || port == "" || players == "" {
        http.Error(w, "Missing parameters", http.StatusBadRequest)
        return
    }

    portInt, _ := strconv.Atoi(port)
    playersInt, _ := strconv.Atoi(players)

    ws := WorldServer{
        IP:       ip,
        Port:     portInt,
        Players:  playersInt,
        RealmID:  id,
        LastSeen: time.Now().Unix(),
    }

    data, _ := json.Marshal(ws)
    rdb.HSet(ctx, "worldservers", id, data)
    w.Write([]byte("OK"))
}

func getLeastLoaded(w http.ResponseWriter, r *http.Request) {
    entries, err := rdb.HGetAll(ctx, "worldservers").Result()
    if err != nil || len(entries) == 0 {
        http.Error(w, "No realms available", http.StatusServiceUnavailable)
        return
    }

    servers := []WorldServer{}
    for _, val := range entries {
        var ws WorldServer
        json.Unmarshal([]byte(val), &ws)
        servers = append(servers, ws)
    }

    sort.Slice(servers, func(i, j int) bool {
        return servers[i].Players < servers[j].Players
    })

    json.NewEncoder(w).Encode(servers[0])
}

func main() {
    initRedis()

    http.HandleFunc("/register", registerWorldServer)
    http.HandleFunc("/least-loaded", getLeastLoaded)

    fmt.Println("Realm registry listening on port 4100")
    log.Fatal(http.ListenAndServe(":4100", nil))
}
