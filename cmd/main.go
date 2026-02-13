package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "github.com/gorilla/mux"
    "github.com/tsv-processor/internal/api"
    "github.com/tsv-processor/internal/config"
    "github.com/tsv-processor/internal/db"
    "github.com/tsv-processor/internal/processor"
)

func main() {
    cfg, err := config.LoadConfig("config.yaml")
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    database, err := db.NewMongoDB(&cfg.Database)
    if err != nil {
        log.Fatalf("Failed to connect to MongoDB: %v", err)
    }
    defer database.Close()

    if err := os.MkdirAll(cfg.Watcher.InputDir, 0755); err != nil {
        log.Fatalf("Failed to create input directory: %v", err)
    }
    if err := os.MkdirAll(cfg.Watcher.OutputDir, 0755); err != nil {
        log.Fatalf("Failed to create output directory: %v", err)
    }

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    workerPool := processor.NewWorkerPool(database, &cfg.Watcher)
    workerPool.Start(ctx)

    handler := api.NewHandler(database)
    router := mux.NewRouter()
    handler.RegisterRoutes(router)

    router.Use(loggingMiddleware)
    router.Use(corsMiddleware)

    srv := &http.Server{
        Addr:         fmt.Sprintf("%s:%d", cfg.API.Host, cfg.API.Port),
        Handler:      router,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    go func() {
        log.Printf("Starting API server on %s:%d", cfg.API.Host, cfg.API.Port)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Failed to start server: %v", err)
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    log.Println("Shutting down server...")
    
    ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancelShutdown()
    
    if err := srv.Shutdown(ctxShutdown); err != nil {
        log.Fatalf("Server forced to shutdown: %v", err)
    }
    
    log.Println("Server stopped")
}

func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Printf("%s %s", r.Method, r.RequestURI)
        next.ServeHTTP(w, r)
    })
}

func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }
        
        next.ServeHTTP(w, r)
    })
}