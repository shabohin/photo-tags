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
	"github.com/shabohin/photo-tags/services/dashboard/internal/api"
	"github.com/shabohin/photo-tags/services/dashboard/internal/config"
	"github.com/shabohin/photo-tags/services/dashboard/internal/metrics"
)

func main() {
	// Загружаем конфигурацию
	cfg := config.Load()

	// Создаем сервис метрик
	metricsService := metrics.NewService(cfg.RabbitMQURL)

	// Создаем HTTP handler
	handler := api.NewHandler(cfg, metricsService)

	// Настраиваем роутер
	router := mux.NewRouter()
	handler.SetupRoutes(router)

	// Создаем HTTP сервер
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Запускаем сервер в отдельной горутине
	go func() {
		log.Printf("Dashboard server starting on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}

func init() {
	// Настройка логирования
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(os.Stdout)

	// Вывод информации о версии
	fmt.Println("Photo Tags Dashboard Service")
	fmt.Println("Version: 1.0.0")
}
