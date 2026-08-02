package database

import (
	"strings"
	"testing"
	"time"

	"new-pay/internal/config"
)

func TestBuildDSN_IncludesTimeoutOptions(t *testing.T) {
	dsn := buildDSN(&config.DatabaseConfig{
		Host:             "localhost",
		Port:             "5432",
		User:             "u",
		Password:         "p",
		Name:             "db",
		SSLMode:          "disable",
		StatementTimeout: 30 * time.Second,
		LockTimeout:      10 * time.Second,
		IdleInTxTimeout:  60 * time.Second,
	})

	for _, want := range []string{
		"options='",
		"-c statement_timeout=30000",
		"-c lock_timeout=10000",
		"-c idle_in_transaction_session_timeout=60000",
	} {
		if !strings.Contains(dsn, want) {
			t.Errorf("DSN missing %q\ngot: %s", want, dsn)
		}
	}
}

func TestBuildDSN_NoOptionsWhenDisabled(t *testing.T) {
	dsn := buildDSN(&config.DatabaseConfig{
		Host: "localhost", Port: "5432", User: "u", Password: "p", Name: "db", SSLMode: "disable",
	})
	if strings.Contains(dsn, "options=") {
		t.Errorf("expected no options when timeouts are zero, got: %s", dsn)
	}
}
