package internal

import (
	"strings"
	"testing"
)

func TestSetupTracingDisabled(t *testing.T) {
	t.Setenv(otelEndpointEnv, "")

	shutdown, err := SetupTracing(t.Context(), false)
	if err != nil {
		t.Fatalf("expected tracing setup to be disabled without an endpoint, got %v", err)
	}

	if err := shutdown(t.Context()); err != nil {
		t.Fatalf("expected disabled tracing shutdown to succeed, got %v", err)
	}
}

func TestSetupTracingRequiresEndpoint(t *testing.T) {
	t.Setenv(otelEndpointEnv, "  ")

	_, err := SetupTracing(t.Context(), true)
	if err == nil {
		t.Fatal("expected tracing setup to require an OTLP endpoint")
	}

	if !strings.Contains(err.Error(), otelEndpointEnv) {
		t.Fatalf("expected error to mention %s, got %v", otelEndpointEnv, err)
	}
}
