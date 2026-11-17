package db

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var Conn = sync.OnceValues(Open)

// Open creates a new connection pool from the POSTGRES_* variables.
// The app uses Conn; Open is for callers that need their own (e.g. tests).
func Open() (*gorm.DB, error) {
	dsn, err := dsnFromEnv()
	if err != nil {
		return nil, err
	}

	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	return gormDB, nil
}

// dsnFromEnv builds the Postgres URL from the POSTGRES_* variables.
// net/url escapes special characters in the user and password.
func dsnFromEnv() (string, error) {
	var missing []string
	get := func(key string) string {
		v := os.Getenv(key)
		if v == "" {
			missing = append(missing, key)
		}
		return v
	}

	host := get("POSTGRES_HOST")
	port := get("POSTGRES_PORT")
	user := get("POSTGRES_USER")
	password := get("POSTGRES_PASSWORD")
	name := get("POSTGRES_DB")
	if len(missing) > 0 {
		return "", fmt.Errorf("missing environment variables: %s", strings.Join(missing, ", "))
	}

	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(user, password),
		Host:     net.JoinHostPort(host, port),
		Path:     name,
		RawQuery: "sslmode=disable",
	}
	return u.String(), nil
}
