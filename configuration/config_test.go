package configuration

import (
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg == nil {
		t.Fatal("expected non-nil default configuration")
	}

	if cfg.Port != "8080" {
		t.Errorf("expected default Port '8080', got '%s'", cfg.Port)
	}

	if cfg.StreamURL != "https://stream.upfluence.co/stream" {
		t.Errorf("expected default StreamURL 'https://stream.upfluence.co/stream', got '%s'", cfg.StreamURL)
	}

	if cfg.EventBusBufferSize != 100 {
		t.Errorf("expected default EventBusBufferSize 100, got %d", cfg.EventBusBufferSize)
	}
}

func TestNew_WithoutEnv(t *testing.T) {
	t.Setenv(EnvPort, "")
	t.Setenv(EnvStreamURL, "")
	t.Setenv(EnvEventBusBufferSize, "")

	cfg := New()
	if cfg == nil {
		t.Fatal("expected non-nil config from New()")
	}

	if cfg.Port != DefaultPort {
		t.Errorf("expected Port '%s', got '%s'", DefaultPort, cfg.Port)
	}

	if cfg.StreamURL != DefaultStreamURL {
		t.Errorf("expected StreamURL '%s', got '%s'", DefaultStreamURL, cfg.StreamURL)
	}

	if cfg.EventBusBufferSize != DefaultEventBusBufferSize {
		t.Errorf("expected EventBusBufferSize %d, got %d", DefaultEventBusBufferSize, cfg.EventBusBufferSize)
	}
}

func TestLoad_WithEnv(t *testing.T) {
	t.Setenv(EnvPort, "9090")
	t.Setenv(EnvStreamURL, "https://custom.stream/events")
	t.Setenv(EnvEventBusBufferSize, "250")

	cfg := Load()
	if cfg == nil {
		t.Fatal("expected non-nil config from Load()")
	}

	if cfg.Port != "9090" {
		t.Errorf("expected Port '9090', got '%s'", cfg.Port)
	}

	if cfg.StreamURL != "https://custom.stream/events" {
		t.Errorf("expected StreamURL 'https://custom.stream/events', got '%s'", cfg.StreamURL)
	}

	if cfg.EventBusBufferSize != 250 {
		t.Errorf("expected EventBusBufferSize 250, got %d", cfg.EventBusBufferSize)
	}
}

func TestLoad_WithInvalidBufferSize(t *testing.T) {
	// Negative or non-numeric values should keep default
	t.Setenv(EnvEventBusBufferSize, "invalid_num")
	cfg1 := Load()
	if cfg1.EventBusBufferSize != DefaultEventBusBufferSize {
		t.Errorf("expected fallback to default buffer size, got %d", cfg1.EventBusBufferSize)
	}

	t.Setenv(EnvEventBusBufferSize, "-5")
	cfg2 := Load()
	if cfg2.EventBusBufferSize != DefaultEventBusBufferSize {
		t.Errorf("expected fallback to default buffer size for negative value, got %d", cfg2.EventBusBufferSize)
	}
}

func TestConfig_Addr(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *Configuration
		expected string
	}{
		{
			name:     "nil config",
			cfg:      nil,
			expected: ":8080",
		},
		{
			name:     "empty port",
			cfg:      &Configuration{Port: ""},
			expected: ":8080",
		},
		{
			name:     "port without colon",
			cfg:      &Configuration{Port: "8080"},
			expected: ":8080",
		},
		{
			name:     "port with colon",
			cfg:      &Configuration{Port: ":9090"},
			expected: ":9090",
		},
		{
			name:     "host and port",
			cfg:      &Configuration{Port: "127.0.0.1:3000"},
			expected: ":127.0.0.1:3000",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.cfg.Addr()
			if got != tc.expected {
				t.Errorf("expected addr '%s', got '%s'", tc.expected, got)
			}
		})
	}
}
