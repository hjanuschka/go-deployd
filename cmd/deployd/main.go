package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hjanuschka/go-deployd/internal/database"
	"github.com/hjanuschka/go-deployd/internal/server"
	"github.com/urfave/cli/v2"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	app := &cli.App{
		Name:  "deployd",
		Usage: "A high-performance, modern reimagining of Deployd in Go",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "port",
				Value:   2403,
				Usage:   "server port",
				EnvVars: []string{"PORT"},
			},
			&cli.StringFlag{
				Name:    "db-type",
				Value:   "mongodb",
				Usage:   "database type (mongodb, sqlite, mysql, postgres)",
				EnvVars: []string{"DB_TYPE"},
			},
			&cli.StringFlag{
				Name:    "db-host",
				Value:   "localhost",
				Usage:   "database host",
				EnvVars: []string{"DB_HOST"},
			},
			&cli.IntFlag{
				Name:    "db-port",
				Value:   0,
				Usage:   "database port (0 = use default for db-type)",
				EnvVars: []string{"DB_PORT"},
			},
			&cli.StringFlag{
				Name:    "db-name",
				Value:   "deployd",
				Usage:   "database name",
				EnvVars: []string{"DB_NAME"},
			},
			&cli.StringFlag{
				Name:    "db-user",
				Usage:   "database username",
				EnvVars: []string{"DB_USER"},
			},
			&cli.StringFlag{
				Name:    "db-pass",
				Usage:   "database password",
				EnvVars: []string{"DB_PASS"},
			},
			&cli.BoolFlag{
				Name:    "db-ssl",
				Usage:   "enable SSL for database connection",
				EnvVars: []string{"DB_SSL"},
			},
			&cli.StringFlag{
				Name:    "config",
				Usage:   "configuration file path",
				EnvVars: []string{"CONFIG_PATH"},
			},
			&cli.BoolFlag{
				Name:    "dev",
				Usage:   "development mode",
				EnvVars: []string{"DEV_MODE"},
			},
		},
		Action: func(c *cli.Context) error {
			return startServer(c)
		},
		Commands: []*cli.Command{
			{
				Name:  "start",
				Usage: "Starts the deployd server (default command)",
				Action: func(c *cli.Context) error {
					return startServer(c)
				},
			},
			{
				Name:      "create-collection",
				Aliases:   []string{"cc"},
				Usage:     "Creates a new collection",
				ArgsUsage: "[collection-name]",
				Action: func(c *cli.Context) error {
					collectionName := c.Args().First()
					if collectionName == "" {
						return cli.NewExitError("Error: Collection name is required", 1)
					}

					dirPath := fmt.Sprintf("resources/%s", collectionName)
					if _, err := os.Stat(dirPath); !os.IsNotExist(err) {
						return cli.NewExitError(fmt.Sprintf("Error: Collection '%s' already exists", collectionName), 1)
					}

					if err := os.MkdirAll(dirPath, 0755); err != nil {
						return cli.NewExitError(fmt.Sprintf("Error creating directory: %v", err), 1)
					}

					configContent := `{
  "properties": {
    "name": {
      "type": "string",
      "required": true
    }
  }
}`
					configPath := fmt.Sprintf("%s/config.json", dirPath)
					if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
						return cli.NewExitError(fmt.Sprintf("Error writing config file: %v", err), 1)
					}

					fmt.Printf("✅ Collection '%s' created successfully at %s\n", collectionName, dirPath)
					return nil
				},
			},
			{
				Name:    "create-user",
				Aliases: []string{"cu"},
				Usage:   "Creates a new user",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "username", Aliases: []string{"u"}, Required: true, Usage: "Username for the new user"},
					&cli.StringFlag{Name: "password", Aliases: []string{"p"}, Required: true, Usage: "Password for the new user"},
					&cli.StringFlag{Name: "email", Aliases: []string{"e"}, Usage: "Email for the new user"},
					&cli.StringFlag{Name: "role", Value: "user", Usage: "Role for the new user"},
				},
				Action: func(c *cli.Context) error {
					return createUser(c)
				},
			},
			{
				Name:    "set-secret",
				Aliases: []string{"ss"},
				Usage:   "Sets a secret in the security configuration",
				ArgsUsage: "[key] [value]",
				Action: func(c *cli.Context) error {
					key := c.Args().Get(0)
					value := c.Args().Get(1)

					if key == "" || value == "" {
						return cli.NewExitError("Error: Both key and value are required", 1)
					}

					return setSecret(key, value)
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func startServer(c *cli.Context) error {
	port := c.Int("port")
	dbType := c.String("db-type")
	dbHost := c.String("db-host")
	dbPort := c.Int("db-port")
	dbName := c.String("db-name")
	dbUser := c.String("db-user")
	dbPass := c.String("db-pass")
	dbSSL := c.Bool("db-ssl")
	config := c.String("config")
	dev := c.Bool("dev")

	// Set default ports based on database type
	if dbPort == 0 {
		switch dbType {
		case "mongodb":
			dbPort = 27017
		case "mysql":
			dbPort = 3306
		case "postgres":
			dbPort = 5432
		case "sqlite":
			dbPort = 0 // SQLite doesn't use ports
		}
	}

	fmt.Printf("🚀 Starting go-deployd server...\n")
	fmt.Printf("   Port: %d\n", port)
	if dbType == "sqlite" {
		fmt.Printf("   Database: %s (SQLite file: %s)\n", dbType, dbName)
	} else {
		fmt.Printf("   Database: %s://%s:%d/%s\n", dbType, dbHost, dbPort, dbName)
	}
	if dev {
		fmt.Printf("   Mode: development\n")
	}

	srv, err := server.New(&server.Config{
		Port:             port,
		DatabaseType:     dbType,
		DatabaseHost:     dbHost,
		DatabasePort:     dbPort,
		DatabaseName:     dbName,
		DatabaseUsername: dbUser,
		DatabasePassword: dbPass,
		DatabaseSSL:      dbSSL,
		ConfigPath:       config,
		Development:      dev,
	})
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("Failed to create server: %v", err), 1)
	}

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      srv,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		fmt.Printf("🌐 Server listening on http://localhost:%d\n", port)
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
	return nil
}

func createUser(c *cli.Context) error {
	dbType := c.String("db-type")
	dbPort := c.Int("db-port")

	if dbPort == 0 {
		switch dbType {
		case "mongodb":
			dbPort = 27017
		case "mysql":
			dbPort = 3306
		case "postgres":
			dbPort = 5432
		case "sqlite":
			dbPort = 0 // SQLite doesn't use ports
		}
	}

	db, err := database.NewDatabase(database.DatabaseType(dbType), &database.Config{
		Host:     c.String("db-host"),
		Port:     dbPort,
		Name:     c.String("db-name"),
		Username: c.String("db-user"),
		Password: c.String("db-pass"),
		SSL:      c.Bool("db-ssl"),
	})
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("Failed to connect to database: %v", err), 1)
	}
	defer db.Close()

	userStore := db.CreateStore("users")

	username := c.String("username")
	password := c.String("password")
	email := c.String("email")
	role := c.String("role")

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return cli.NewExitError("Failed to hash password", 1)
	}

	user := map[string]interface{}{
		"username": username,
		"password": string(hashedPassword),
		"email":    email,
		"role":     role,
	}

	_, err = userStore.Insert(context.Background(), user)
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("Failed to create user: %v", err), 1)
	}

	fmt.Printf("✅ User '%s' created successfully.\n", username)
	return nil
}

func setSecret(key, value string) error {
	configPath := ".deployd/security.json"
	file, err := os.ReadFile(configPath)
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("Failed to read security config: %v", err), 1)
	}

	var securityConfig map[string]interface{}
	if err := json.Unmarshal(file, &securityConfig); err != nil {
		return cli.NewExitError(fmt.Sprintf("Failed to parse security config: %v", err), 1)
	}

	securityConfig[key] = value

	newConfig, err := json.MarshalIndent(securityConfig, "", "  ")
	if err != nil {
		return cli.NewExitError(fmt.Sprintf("Failed to marshal new security config: %v", err), 1)
	}

	if err := os.WriteFile(configPath, newConfig, 0600); err != nil {
		return cli.NewExitError(fmt.Sprintf("Failed to write security config: %v", err), 1)
	}

	fmt.Printf("✅ Secret '%s' set successfully.\n", key)
	return nil
}