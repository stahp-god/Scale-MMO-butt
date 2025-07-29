
/*
==============================
PHASE 7: ASYNC DB WRITE DAEMON
==============================

This Go service receives write events (e.g. character updates) via Redis queue,
and flushes them in batch to MySQL to reduce write pressure on the master.

------------------------------
🛠 How to Run This Writer
------------------------------

1. Install Go and MySQL client driver:
   sudo apt install golang
   go get github.com/go-redis/redis/v8
   go get github.com/go-sql-driver/mysql

2. Ensure Redis and MySQL (mysql-master) are running

3. Save this file as: async_writer.go

4. Build:
   go build -o async-writer async_writer.go

5. Run:
   ./async-writer

------------------------------
📌 What It Does
------------------------------
- Pulls write instructions from Redis list: write_queue
- Writes them to MySQL in batch every 3 seconds
- Decreases DB contention under load
*/

package main

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "time"

    "github.com/go-redis/redis/v8"
    _ "github.com/go-sql-driver/mysql"
)

var (
    ctx       = context.Background()
    redisList = "write_queue"
)

type WriteRequest struct {
    SQL  string   `json:"sql"`
    Args []string `json:"args"`
}

func main() {
    rdb := redis.NewClient(&redis.Options{
        Addr:     "localhost:6379",
        Password: "",
        DB:       0,
    })

    dsn := "trinity:trinitypass@tcp(localhost:3306)/trinity"
    db, err := sql.Open("mysql", dsn)
    if err != nil {
        log.Fatalf("Failed to connect to MySQL: %v", err)
    }

    for {
        batch := []WriteRequest{}

        for i := 0; i < 10; i++ {
            val, err := rdb.LPop(ctx, redisList).Result()
            if err == redis.Nil {
                break
            } else if err != nil {
                log.Printf("Redis error: %v", err)
                break
            }

            var req WriteRequest
            if err := json.Unmarshal([]byte(val), &req); err == nil {
                batch = append(batch, req)
            }
        }

        if len(batch) > 0 {
            tx, err := db.Begin()
            if err != nil {
                log.Printf("DB transaction error: %v", err)
                continue
            }

            for _, req := range batch {
                args := make([]interface{}, len(req.Args))
                for i, v := range req.Args {
                    args[i] = v
                }

                _, err := tx.Exec(req.SQL, args...)
                if err != nil {
                    log.Printf("Exec failed: %v", err)
                }
            }

            tx.Commit()
            log.Printf("Flushed %d writes", len(batch))
        }

        time.Sleep(3 * time.Second)
    }
}
