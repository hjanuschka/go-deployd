package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hjanuschka/go-deployd/internal/server"
)

func main() {
	var (
		port    = flag.Int("port", 2403, "server port")
		dbHost  = flag.String("db-host", "localhost", "MongoDB host")
		dbPort  = flag.Int("db-port", 27017, "MongoDB port")
		dbName  = flag.String("db-name", "deployd", "MongoDB database name")
		config  = flag.String("config", "", "configuration file path")
		dev     = flag.Bool("dev", false, "development mode")
	)
	flag.Parse()

	fmt.Printf("🚀 Starting go-deployd server...\n")
	fmt.Printf("   Port: %d\n", *port)
	fmt.Printf("   Database: %s:%d/%s\n", *dbHost, *dbPort, *dbName)
	if *dev {
		fmt.Printf("   Mode: development\n")
	}

	srv, err := server.New(&server.Config{
		Port:           *port,
		DatabaseHost:   *dbHost,
		DatabasePort:   *dbPort,
		DatabaseName:   *dbName,
		ConfigPath:     *config,
		Development:    *dev,
	})
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: srv,
	}

	go func() {
		fmt.Printf("🌐 Server listening on http://localhost:%d\n", *port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n🛑 Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	if err := srv.Close(); err != nil {
		log.Printf("Error closing server resources: %v", err)
	}

	fmt.Println("✅ Server shutdown complete")
}