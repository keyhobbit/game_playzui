package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"github.com/game-playzui/tienlen-server/internal/auth"
	"github.com/game-playzui/tienlen-server/internal/bot"
	"github.com/game-playzui/tienlen-server/internal/config"
	"github.com/game-playzui/tienlen-server/internal/game"
	"github.com/game-playzui/tienlen-server/internal/handlers"
	"github.com/game-playzui/tienlen-server/internal/matchmaking"
	"github.com/game-playzui/tienlen-server/internal/repository"
	"github.com/game-playzui/tienlen-server/internal/ws"
)

func main() {
	cfg := config.Load()

	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}
	log.Println("connected to PostgreSQL")

	if err := repository.RunMigrations(db); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPwd,
		DB:       0,
	})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("redis ping failed: %v", err)
	}
	log.Println("connected to Redis")

	userRepo := repository.NewUserRepo(db)
	jwtService := auth.NewJWTService(cfg.JWTSecret)

	hub := ws.NewHub()
	go hub.Run()

	mm := matchmaking.NewService(rdb, hub)
	go mm.Start()

	_ = game.NewEngine(hub, mm)

	botManager := bot.NewManager(hub)
	go botManager.Run()

	authHandler := handlers.NewAuthHandler(userRepo, jwtService)
	roomHandler := handlers.NewRoomHandler(hub, mm)
	userHandler := handlers.NewUserHandler(userRepo)
	wsHandler := handlers.NewWSHandler(hub, jwtService, userRepo, mm)

	r := mux.NewRouter()
	r.Use(corsMiddleware)

	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/auth/register", authHandler.Register).Methods("POST", "OPTIONS")
	api.HandleFunc("/auth/login", authHandler.Login).Methods("POST", "OPTIONS")

	protected := api.PathPrefix("").Subrouter()
	protected.Use(auth.Middleware(jwtService))
	protected.HandleFunc("/user/profile", userHandler.Profile).Methods("GET", "OPTIONS")
	protected.HandleFunc("/rooms", roomHandler.ListRooms).Methods("GET", "OPTIONS")

	r.HandleFunc("/ws", wsHandler.HandleUpgrade)

	r.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	}).Methods("GET")

	addr := fmt.Sprintf("0.0.0.0:%d", cfg.AppPort)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("server listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown failed: %v", err)
	}
	log.Println("server stopped")
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
