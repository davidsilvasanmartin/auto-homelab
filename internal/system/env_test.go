package system

import (
	"errors"
	"log"
	"strings"
	"testing"
)

type mockEnvLookup struct {
	lookupEnvFunc func(key string) (string, bool)
}

func (m *mockEnvLookup) lookupEnv(key string) (string, bool) {
	if m.lookupEnvFunc != nil {
		return m.lookupEnvFunc(key)
	}
	return "", false
}

func TestDefaultEnv_GetEnv_ReturnsLookupEnv(t *testing.T) {
	value := "my_value"
	exists := true
	mock := &mockEnvLookup{
		lookupEnvFunc: func(key string) (string, bool) {
			return value, exists
		},
	}
	env := &DefaultEnv{LookupEnv: mock.lookupEnv}

	gotValue, gotExists := env.GetEnv("DB_URL")

	if gotExists != exists {
		t.Fatalf("expected exists to be %v, got %v", exists, gotExists)
	}
	if gotValue != value {
		t.Errorf("expected var value %q, got %q", value, gotValue)
	}
}

func TestDefaultEnv_GetRequiredEnv_Success(t *testing.T) {
	varValue := "localhost:5432/db"
	mock := &mockEnvLookup{
		lookupEnvFunc: func(key string) (string, bool) {
			return varValue, true
		},
	}
	env := &DefaultEnv{LookupEnv: mock.lookupEnv}

	obtainedValue, err := env.GetRequiredEnv("DB_URL")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if obtainedValue != varValue {
		t.Errorf("expected varValue %q, got %q", varValue, obtainedValue)
	}
}

func TestDefaultEnv_GetRequiredEnv_NotFound(t *testing.T) {
	varName := "MISSING_VAR"
	mock := &mockEnvLookup{
		lookupEnvFunc: func(key string) (string, bool) {
			// Simulate env variable not being set
			return "", false
		},
	}
	env := &DefaultEnv{LookupEnv: mock.lookupEnv}

	_, err := env.GetRequiredEnv(varName)

	if err == nil {
		log.Fatal("expected error when env var is missing, got nil")
	}
	if !errors.Is(err, ErrRequiredEnvNotFound) {
		t.Errorf("expected ErrRequiredEnvNotFound, got: %v", err)
	}
	if !strings.Contains(err.Error(), varName) {
		t.Errorf("expected error message to contain var name %q, got: %s", varName, err.Error())
	}
}

func TestDefaultEnv_GetRequiredEnv_EmptyValue(t *testing.T) {
	mock := &mockEnvLookup{
		lookupEnvFunc: func(key string) (string, bool) {
			return "", true
		},
	}
	env := &DefaultEnv{LookupEnv: mock.lookupEnv}

	value, err := env.GetRequiredEnv("EMPTY_VAR")

	if err != nil {
		t.Errorf("expected no error when variable exists with empty value, got: %v", err)
	}
	if value != "" {
		t.Errorf("expected empty value, got %q", value)
	}
}
