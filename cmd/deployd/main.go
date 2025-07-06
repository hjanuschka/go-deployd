package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/hjanuschka/go-deployd/internal/server"
)

func main() {
	var (
		port       = flag.Int("port", 2403, "Server port")
		dbType     = flag.String("db-type", "mongodb", "Database type (mongodb, sqlite, mysql, postgres)")
		dbHost     = flag.String("db-host", "localhost", "Database host")
		dbPort     = flag.Int("db-port", 27017, "Database port")
		dbName     = flag.String("db-name", "deployd", "Database name")
		dbUsername = flag.String("db-username", "", "Database username")
		dbPassword = flag.String("db-password", "", "Database password")
		dbSSL      = flag.Bool("db-ssl", false, "Use SSL for database connection")
		configPath = flag.String("config", "./", "Path to configuration directory")
		dev        = flag.Bool("dev", false, "Development mode")
	)

	flag.Parse()

	config := &server.Config{
		Port:             *port,
		DatabaseType:     *dbType,
		DatabaseHost:     *dbHost,
		DatabasePort:     *dbPort,
		DatabaseName:     *dbName,
		DatabaseUsername: *dbUsername,
		DatabasePassword: *dbPassword,
		DatabaseSSL:      *dbSSL,
		ConfigPath:       *configPath,
		Development:      *dev,
	}

	srv, err := server.New(config)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	fmt.Printf("🚀 go-deployd starting on port %d\n", config.Port)
	fmt.Printf("📊 Database: %s\n", config.DatabaseType)
	if config.Development {
		fmt.Println("🔧 Development mode enabled")
	}

	addr := fmt.Sprintf(":%d", config.Port)
	fmt.Printf("🌐 Server listening on http://localhost%s\n", addr)
	
	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}