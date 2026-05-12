package main

import (
	"context"
	"errors"
	"log/slog" // Используем современный логер
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	deptsvc "department-api/internal/department/services"
	employeesvc "department-api/internal/employee/services"
	"department-api/internal/httpserver"
	"department-api/internal/persistence"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Настраиваем структурированный логер (JSON для продакшена, Text для разработки)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}

	store := persistence.NewStore(db)
	deptSvc := deptsvc.New(db, store)
	empSvc := employeesvc.New(db, store, store)

	mux := http.NewServeMux()
	httpserver.RegisterRoutes(mux, deptSvc, empSvc)

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("listening on", "addr", addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("listen and serve failed", "error", err)
			os.Exit(1)
		}
	}()

	<-stop
	slog.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("server exited cleanly")
}
