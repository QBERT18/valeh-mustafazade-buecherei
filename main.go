package main

import (
	"context"
	"log"
	"os"

	"github.com/QBERT18/valeh-mustafazade-buecherei/api"
	"github.com/QBERT18/valeh-mustafazade-buecherei/db"
	"github.com/QBERT18/valeh-mustafazade-buecherei/seed"
	"github.com/QBERT18/valeh-mustafazade-buecherei/storage"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env for local `go run`. Missing file is fine (e.g. in Docker), and
	// variables already set in the environment are not overridden.
	_ = godotenv.Load()

	// Open the database connection once at startup so the app fails fast if it is unreachable.
	g, err := db.Conn()
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	if err := db.Migrate(g); err != nil {
		log.Fatalf("database migration failed: %v", err)
	}
	st, err := storage.Conn()
	if err != nil {
		log.Fatalf("storage connection failed: %v", err)
	}

	// `buecherei seed` (or `go run . seed`) fills the database with sample data and exits.
	if len(os.Args) > 1 && os.Args[1] == "seed" {
		if err := seed.Run(context.Background(), g, st); err != nil {
			log.Fatalf("seeding failed: %v", err)
		}
		return
	}

	// Create a Gin router with default middleware (logger and recovery)
	r := gin.Default()
	api.Register(r, g, st)

	// Start server on port 8080 (default)
	// Server will listen on 0.0.0.0:8080 (localhost:8080 on Windows)
	r.Run()
}
