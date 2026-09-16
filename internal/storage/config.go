package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func SetConfig(ctx context.Context, pool *pgxpool.Pool, namespace string, key string, value string) error {
	query := `
            INSERT INTO configs (namespace, key, value, version, updated_at)
            VALUES ($1, $2, $3, 1, NOW())
            ON CONFLICT (namespace, key)
            DO UPDATE SET
                value = EXCLUDED.value,
                version = configs.version + 1,
                updated_at = NOW();
          `

	_, err := pool.Exec(ctx, query, namespace, key, value)
	if err != nil {
		return fmt.Errorf("failed to set config: %w", err)
	}
	return nil
}

func GetConfig(ctx context.Context, pool *pgxpool.Pool, namespace string, key string) (string, error) {
	query := `
            SELECT value FROM configs WHERE namespace = $1 AND key = $2;
          `

	var value string

	err := pool.QueryRow(ctx, query, namespace, key).Scan(&value)
	if err != nil {
		return "", err
	}

	return value, nil
}
