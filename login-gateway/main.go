
/*
==============================
PHASE 8: LOGIN GATEWAY WITH METRICS
==============================

This version of the login gateway includes a Prometheus /metrics endpoint.

------------------------------
🛠 How to Run
------------------------------
1. Build:
   go build -o login-gateway-metrics main.go

2. Run:
   ./login-gateway-metrics

3. Prometheus will scrape metrics from :9100/metrics

Exports:
- login_attempts_total
- active_queue_size
*/

package main

import (
    "fmt"
    "io"
    "log"
    "net"
    "net/http"
    "os"
    "strings"
    "sync"
    "time"

    "github.com/go-redis/redis/v8"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "golang.org/x/net/context"
)

var (
    ctx        = context.Background()
    redisQueue = "login_queue"
    rdb        *redis.Client
    queueLock  sync.Mutex
    maxQueue   = 5000
    rateLimit  = time.Second / 2
    authServer = "127.0.0.1:3725"

    loginAttempts = prometheus.NewCounter(prometheus.CounterOpts{
        Name: "login_attempts_total",
        Help: "Total login connection attempts",
    })

    queueSize = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "active_queue_size",
        Help: "Current number of clients in queue",
    })
)

func initRedis() {
    rdb = redis.NewClient(&redis.Options{
        Addr:     "localhost:6379",
        Password: "",
        DB:       0,
    })
    _, err := rdb.Ping(ctx).Result()
    if err != nil {
        log.Fatalf("Failed to connect to Redis: %v", err)
    }
}

func handleClient(conn net.Conn) {
    defer conn.Close()
    id := fmt.Sprintf("%s-%d", conn.RemoteAddr().String(), time.Now().UnixNano())

    loginAttempts.Inc()
    qlen, _ := rdb.LLen(ctx, redisQueue).Result()
    queueSize.Set(float64(qlen))

    if qlen >= int64(maxQueue) {
        conn.Write([]byte("Server is full. Try again later.
"))
        return
    }

    rdb.LPush(ctx, redisQueue, id)
    log.Printf("Enqueued %s", id)

    for {
        head, err := rdb.RPop(ctx, redisQueue).Result()
        if err == redis.Nil {
            time.Sleep(100 * time.Millisecond)
            continue
        }
        if head == id {
            forwardTraffic(conn)
            return
        } else {
            rdb.RPush(ctx, redisQueue, head)
            time.Sleep(500 * time.Millisecond)
        }
    }
}

func forwardTraffic(client net.Conn) {
    backend, err := net.Dial("tcp", authServer)
    if err != nil {
        log.Printf("Authserver unreachable: %v", err)
        client.Write([]byte("Authserver is down. Try later.
"))
        return
    }
    defer backend.Close()

    go io.Copy(backend, client)
    io.Copy(client, backend)
}

func startMetrics() {
    prometheus.MustRegister(loginAttempts)
    prometheus.MustRegister(queueSize)

    http.Handle("/metrics", promhttp.Handler())
    go func() {
        log.Println("Metrics listening on :9100")
        log.Fatal(http.ListenAndServe(":9100", nil))
    }()
}

func main() {
    initRedis()
    startMetrics()

    port := "3724"
    listener, err := net.Listen("tcp", ":"+port)
    if err != nil {
        log.Fatalf("Failed to bind on %s: %v", port, err)
    }
    defer listener.Close()

    log.Printf("Login gateway listening on port %s", port)
    for {
        conn, err := listener.Accept()
        if err != nil {
            log.Printf("Connection error: %v", err)
            continue
        }
        go handleClient(conn)
    }
}
