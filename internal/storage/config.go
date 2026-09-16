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

func ListConfigs(ctx context.Context, pool *pgxpool.Pool, namespace string) (map[string]string, error) {
	query := `
            SELECT key, value FROM configs WHERE namespace = $1;
          `

	rows, err := pool.Query(ctx, query, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to query configs: %w", err)
	}
	defer rows.Close()

	result := make(map[string]string)

	for rows.Next() {
		var key, value string

		err := rows.Scan(&key, &value)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		result[key] = value
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return result, nil
}

func DeleteConfig(ctx context.Context, pool *pgxpool.Pool, namespace, key string) (int64, error) {
    query := `DELETE FROM configs WHERE namespace = $1 AND key = $2;`

    exec, err := pool.Exec(ctx, query, namespace, key)
    if err != nil {
        return 0, fmt.Errorf("delete config: %w", err)
    }

    return exec.RowsAffected(), nil
}
