package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/hjanuschka/go-deployd/internal/logging"
	"github.com/hjanuschka/go-deployd/internal/server"
)

func main() {
	var (
		port   = flag.Int("port", 2403, "server port")
		dbType = flag.String("db-type", "mongodb", "database type (mongodb, sqlite, mysql, postgres)")
		dbHost = flag.String("db-host", "localhost", "database host")
		dbPort = flag.Int("db-port", 0, "database port (0 = use default for db-type)")
		dbName = flag.String("db-name", "deployd", "database name")
		dbUser = flag.String("db-user", "", "database username")
		dbPass = flag.String("db-pass", "", "database password")
		dbSSL  = flag.Bool("db-ssl", false, "enable SSL for database connection")
		config = flag.String("config", "", "configuration file path")
		dev    = flag.Bool("dev", false, "development mode")
	)
	flag.Parse()

	// Set default ports based on database type
	if *dbPort == 0 {
		switch *dbType {
		case "mongodb":
			*dbPort = 27017
		case "mysql":
			*dbPort = 3306
		case "postgres":
			*dbPort = 5432
		case "sqlite":
			*dbPort = 0 // SQLite doesn't use ports
		}
	}

	// Initialize logging early for startup messages
	// Use environment variable for log level, with dev mode override
	logLevel := logging.INFO
	if *dev {
		logLevel = logging.DEBUG
	}
	
	// Check for LOG_LEVEL environment variable override
	if envLevel := os.Getenv("LOG_LEVEL"); envLevel != "" {
		switch strings.ToUpper(envLevel) {
		case "DEBUG":
			logLevel = logging.DEBUG
		case "INFO":
			logLevel = logging.INFO
		case "WARN", "WARNING":
			logLevel = logging.WARN
		case "ERROR":
			logLevel = logging.ERROR
		}
	}
	
	logging.InitializeLogger(logging.Config{
		LogDir:    "./logs",
		DevMode:   *dev,
		MinLevel:  logLevel,
		Component: "main",
	})
	
	logger := logging.GetLogger()
	logger.Info("Starting go-deployd server", logging.Fields{
		"port": *port,
		"database_type": *dbType,
		"development_mode": *dev,
	})
	
	if *dbType == "sqlite" {
		logger.Info("Using SQLite database", logging.Fields{
			"database_file": *dbName,
		})
	} else {
		logger.Info("Using network database", logging.Fields{
			"database_type": *dbType,
			"host": *dbHost,
			"port": *dbPort,
			"database": *dbName,
		})
	}

	// Ensure js-sandbox has npm modules installed for JavaScript events
	checkJSSandboxModules()

	srv, err := server.New(&server.Config{
		Port:             *port,
		DatabaseType:     *dbType,
		DatabaseHost:     *dbHost,
		DatabasePort:     *dbPort,
		DatabaseName:     *dbName,
		DatabaseUsername: *dbUser,
		DatabasePassword: *dbPass,
		DatabaseSSL:      *dbSSL,
		ConfigPath:       *config,
		Development:      *dev,
	})
	
	// For now, skip embedded dashboard - will implement later
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", *port),
		Handler:      srv,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logging.GetLogger().Info("Server listening", logging.Fields{
			"url": fmt.Sprintf("http://localhost:%d", *port),
		})
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logging.GetLogger().Info("Shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	logging.GetLogger().Info("Server gracefully stopped")
	logging.Shutdown()
}

// checkJSSandboxModules ensures npm modules are installed for JavaScript event handlers
func checkJSSandboxModules() {
	jsSandboxDir := "js-sandbox"
	nodeModulesDir := filepath.Join(jsSandboxDir, "node_modules")
	packageJSONPath := filepath.Join(jsSandboxDir, "package.json")

	// Check if js-sandbox directory exists
	if _, err := os.Stat(jsSandboxDir); os.IsNotExist(err) {
		return // No js-sandbox, skip
	}

	// Check if package.json exists
	if _, err := os.Stat(packageJSONPath); os.IsNotExist(err) {
		return // No package.json, skip
	}

	// Check if node_modules exists
	if _, err := os.Stat(nodeModulesDir); os.IsNotExist(err) {
		logging.GetLogger().Info("Installing JavaScript event handler dependencies")
		
		// Run npm install in js-sandbox directory
		cmd := exec.Command("npm", "install")
		cmd.Dir = jsSandboxDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		if err := cmd.Run(); err != nil {
			logging.GetLogger().Warn("Failed to install js-sandbox dependencies", logging.Fields{
				"error": err.Error(),
				"impact": "JavaScript events may not have access to npm modules",
			})
		} else {
			logging.GetLogger().Info("JavaScript event handler dependencies installed")
		}
	}
}
