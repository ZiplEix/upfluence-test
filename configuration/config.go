// Package configuration manages application-wide configuration parameters
// and environment variable loading with fallback to default settings.
package configuration

import (
	"os"
	"strconv"
	"strings"
)

// Default configuration constants matching README.md.
const (
	DefaultPort               = "8080"
	DefaultStreamURL          = "https://stream.upfluence.co/stream"
	DefaultEventBusBufferSize = 100
)

// Environment variable names matching README.md.
const (
	EnvPort               = "PORT"
	EnvStreamURL          = "STREAM_URL"
	EnvEventBusBufferSize = "EVENT_BUS_BUFFER_SIZE"
)

// Configuration represents the application configuration settings.
type Configuration struct {
	Port               string `json:"port"`
	StreamURL          string `json:"stream_url"`
	EventBusBufferSize int    `json:"event_bus_buffer_size"`
}

// Default returns a new Config instance populated with the standard default values.
func Default() *Configuration {
	return &Configuration{
		Port:               DefaultPort,
		StreamURL:          DefaultStreamURL,
		EventBusBufferSize: DefaultEventBusBufferSize,
	}
}

// New initializes and returns a new Config instance loaded from environment variables,
// falling back to default values when variables are unset or empty.
func New() *Configuration {
	return Load()
}

// Load loads configuration from environment variables, falling back to defaults.
func Load() *Configuration {
	cfg := Default()

	if port := os.Getenv(EnvPort); port != "" {
		cfg.Port = strings.TrimSpace(port)
	}

	if streamURL := os.Getenv(EnvStreamURL); streamURL != "" {
		cfg.StreamURL = strings.TrimSpace(streamURL)
	}

	if bufStr := os.Getenv(EnvEventBusBufferSize); bufStr != "" {
		if bufSize, err := strconv.Atoi(strings.TrimSpace(bufStr)); err == nil && bufSize > 0 {
			cfg.EventBusBufferSize = bufSize
		}
	}

	return cfg
}

// Addr returns the network address formatted for http.Server.Addr (e.g. ":8080").
func (c *Configuration) Addr() string {
	if c == nil || c.Port == "" {
		return ":" + DefaultPort
	}

	if strings.HasPrefix(c.Port, ":") {
		return c.Port
	}

	return ":" + c.Port
}
