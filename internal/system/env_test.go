package system

import (
	"errors"
	"log"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestDefaultEnv_GetEnv_ReturnsValueFromViper(t *testing.T) {
	value := "my_value"
	varName := "DB_URL"
	v := viper.New()
	v.Set(varName, value)
	env := &DefaultEnv{
		ViperConfig: func() *viper.Viper { return v },
	}

	gotValue, gotExists := env.GetEnv(varName)

	if !gotExists {
		t.Fatalf("expected exists to be true, got false")
	}
	if gotValue != value {
		t.Errorf("expected var value %q, got %q", value, gotValue)
	}
}

func TestDefaultEnv_GetEnv_VariableNotSet(t *testing.T) {
	varName := "NONEXISTENT_VAR"
	v := viper.New()
	env := &DefaultEnv{
		ViperConfig: func() *viper.Viper { return v },
	}

	gotValue, gotExists := env.GetEnv(varName)

	if gotExists {
		t.Fatalf("expected exists to be false, got true")
	}
	if gotValue != "" {
		t.Errorf("expected empty value when variable doesn't exist, got %q", gotValue)
	}
}

func TestDefaultEnv_GetEnv_PanicsWhenViperConfigIsNil(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic when ViperConfig is nil, but did not panic")
		}
	}()

	env := &DefaultEnv{
		ViperConfig: nil,
	}

	env.GetEnv("SOME_VAR")
}

func TestDefaultEnv_GetRequiredEnv_Success(t *testing.T) {
	varValue := "localhost:5432/db"
	varName := "DB_URL"
	v := viper.New()
	v.Set(varName, varValue)
	env := &DefaultEnv{
		ViperConfig: func() *viper.Viper { return v },
	}

	gotValue, err := env.GetRequiredEnv(varName)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if gotValue != varValue {
		t.Errorf("expected varValue %q, got %q", varValue, gotValue)
	}
}

func TestDefaultEnv_GetRequiredEnv_NotFound(t *testing.T) {
	varName := "MISSING_VAR"
	v := viper.New()
	env := &DefaultEnv{
		ViperConfig: func() *viper.Viper { return v },
	}

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
	varName := "EMPTY_VAR"
	v := viper.New()
	v.Set(varName, "")
	env := &DefaultEnv{
		ViperConfig: func() *viper.Viper { return v },
	}

	value, err := env.GetRequiredEnv(varName)

	if err != nil {
		t.Errorf("expected no error when variable exists with empty value, got: %v", err)
	}
	if value != "" {
		t.Errorf("expected empty value, got %q", value)
	}
}
