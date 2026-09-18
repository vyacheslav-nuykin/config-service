package storage

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/config_service?sslmode=disable"
	}

	ctx := context.Background()

	if err := RunMigrations(dbURL); err != nil {
		panic("failed to run migrations: " + err.Error())
	}

	var err error
	testPool, err = New(ctx, dbURL)
	if err != nil {
		panic("failed to connect to test db: " + err.Error())
	}

	code := m.Run()

	_, _ = testPool.Exec(ctx, "TRUNCATE TABLE configs")
	testPool.Close()

	os.Exit(code)
}

func cleanConfig(t *testing.T, namespace, key string) {
	t.Helper()

	ctx := context.Background()

	_, _ = DeleteConfig(ctx, testPool, namespace, key)
	t.Cleanup(func() {
		_, _ = DeleteConfig(ctx, testPool, namespace, key)
	})
}

func TestSetConfig(t *testing.T) {
	ctx := context.Background()
	namespace := "test_set_ns"
	key := "test_set_key"
	value := "test_value"

	cleanConfig(t, namespace, key)

	if err := SetConfig(ctx, testPool, namespace, key, value); err != nil {
		t.Fatalf("SetConfig failed: %v", err)
	}

	gotValue, err := GetConfig(ctx, testPool, namespace, key)
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if gotValue != value {
		t.Errorf("GetConfig() = %v, want %v", gotValue, value)
	}
}

func TestSetConfigUpdate(t *testing.T) {
	ctx := context.Background()
	namespace := "test_update_ns"
	key := "test_update_key"
	value := "initial_value"
	newValue := "new_value"

	cleanConfig(t, namespace, key)

	if err := SetConfig(ctx, testPool, namespace, key, value); err != nil {
		t.Fatalf("SetConfig failed: %v", err)
	}

	if err := SetConfig(ctx, testPool, namespace, key, newValue); err != nil {
		t.Fatalf("SetConfig update failed: %v", err)
	}

	gotValue, err := GetConfig(ctx, testPool, namespace, key)
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if gotValue != newValue {
		t.Errorf("GetConfig() after update = %v, want %v", gotValue, newValue)
	}
}

func TestGetConfig(t *testing.T) {
	ctx := context.Background()
	namespace := "test_get_ns"
	key := "test_get_key"
	value := "test_value"

	cleanConfig(t, namespace, key)

	if err := SetConfig(ctx, testPool, namespace, key, value); err != nil {
		t.Fatalf("SetConfig failed: %v", err)
	}

	gotValue, err := GetConfig(ctx, testPool, namespace, key)
	if err != nil {
		t.Fatalf("GetConfig failed: %v", err)
	}
	if gotValue != value {
		t.Errorf("GetConfig() = %v, want %v", gotValue, value)
	}
}

func TestGetConfigNotFound(t *testing.T) {
	ctx := context.Background()
	namespace := "test_get_missing_ns"
	key := "non_existent_key"

	cleanConfig(t, namespace, key)

	_, err := GetConfig(ctx, testPool, namespace, key)
	if err == nil {
		t.Errorf("GetConfig() for missing key should return error")
	}
}

func TestListConfigs(t *testing.T) {
	ctx := context.Background()
	namespace := "test_list_ns"
	key := "test_list_key"
	value := "test_value"

	cleanConfig(t, namespace, key)

	if err := SetConfig(ctx, testPool, namespace, key, value); err != nil {
		t.Fatalf("SetConfig failed: %v", err)
	}

	configs, err := ListConfigs(ctx, testPool, namespace)
	if err != nil {
		t.Fatalf("ListConfigs failed: %v", err)
	}
	if configs[key] != value {
		t.Errorf("ListConfigs() missing key, got: %v", configs)
	}
}

func TestListConfigsEmpty(t *testing.T) {
	ctx := context.Background()
	namespace := "test_list_empty_ns"
	cleanConfig(t, namespace, "placeholder")

	configs, err := ListConfigs(ctx, testPool, namespace)
	if err != nil {
		t.Fatalf("ListConfigs failed: %v", err)
	}
	if configs == nil {
		t.Error("expected non-nil map, got nil")
	}
	if len(configs) != 0 {
		t.Errorf("expected empty map, got %d items", len(configs))
	}
}

func TestDeleteConfig(t *testing.T) {
	ctx := context.Background()
	namespace := "test_delete_ns"
	key := "test_delete_key"
	value := "test_value"

	cleanConfig(t, namespace, key)

	if err := SetConfig(ctx, testPool, namespace, key, value); err != nil {
		t.Fatalf("SetConfig failed: %v", err)
	}

	rows, err := DeleteConfig(ctx, testPool, namespace, key)
	if err != nil {
		t.Fatalf("DeleteConfig failed: %v", err)
	}
	if rows == 0 {
		t.Errorf("DeleteConfig() deleted 0 rows, expected 1")
	}

	_, err = GetConfig(ctx, testPool, namespace, key)
	if err == nil {
		t.Errorf("GetConfig() after delete should return error")
	}
}

func TestDeleteConfigNotFound(t *testing.T) {
	ctx := context.Background()
	namespace := "test_delete_missing_ns"
	key := "never_existed"

	cleanConfig(t, namespace, key)

	rows, err := DeleteConfig(ctx, testPool, namespace, key)
	if err != nil {
		t.Fatalf("DeleteConfig failed: %v", err)
	}
	if rows != 0 {
		t.Errorf("expected 0 rows affected, got %d", rows)
	}
}

func TestNewInvalidURL(t *testing.T) {
	ctx := context.Background()
	_, err := New(ctx, "invalid://url:::bad")
	if err == nil {
		t.Error("expected error for invalid URL, got nil")
	}
}

func TestNewUnreachableDB(t *testing.T) {
	ctx := context.Background()
	_, err := New(ctx, "postgres://postgres:postgres@localhost:9999/nope?sslmode=disable")
	if err == nil {
		t.Error("expected error for unreachable DB, got nil")
	}
}
