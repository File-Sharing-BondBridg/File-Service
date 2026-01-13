package infrastructure

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

type PostgresStorage struct {
	Db *sql.DB
}

var postgresShards map[int]*PostgresStorage
var postgresShardCount int

// Connect establishes connection to PostgreSQL
func (p *PostgresStorage) Connect(connectionString string) error {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %v", err)
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping PostgreSQL: %v", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	p.Db = db

	// Create tables
	if err := p.createTables(); err != nil {
		return fmt.Errorf("failed to create tables: %v", err)
	}

	log.Println("Connected to PostgreSQL successfully")
	return nil
}

func InitializePostgresShards(connections []string) error {
	postgresShards = make(map[int]*PostgresStorage)
	postgresShardCount = len(connections)

	for i, conn := range connections {
		pg := &PostgresStorage{}
		if err := pg.Connect(conn); err != nil {
			return fmt.Errorf("failed to connect shard %d: %w", i, err)
		}
		postgresShards[i] = pg
	}

	return nil
}

func GetStats() map[string]interface{} {
	if postgresShards == nil {
		return map[string]interface{}{}
	}

	stats := map[string]interface{}{}
	for i, shard := range postgresShards {
		stats[fmt.Sprintf("shard_%d", i)] = shard.getStats()
	}
	return stats
}

func GetPostgresForUser(userID string) *PostgresStorage {
	shard := ResolveShard(userID, postgresShardCount)
	log.Printf("[DB] user=%s → shard=%d", userID, shard)
	return postgresShards[shard]
}
