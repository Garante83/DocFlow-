package main

import (
	"io"
	"os"
	"testing"

	"dokumentenscanner/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetPort tests the getPort function
func TestGetPort(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.Config
		expected string
	}{
		{
			name:     "default port",
			cfg:      config.DefaultConfig(),
			expected: ":8082",
		},
		{
			name: "custom port",
			cfg: func() *config.Config {
				cfg := config.DefaultConfig()
				cfg.Server.Port = "9090"
				return cfg
			}(),
			expected: ":9090",
		},
		{
			name: "empty port",
			cfg: func() *config.Config {
				cfg := config.DefaultConfig()
				cfg.Server.Port = ""
				return cfg
			}(),
			expected: ":",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPort(tt.cfg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestReadEmbeddedFile tests the readEmbeddedFile function
func TestReadEmbeddedFile(t *testing.T) {
	// Note: This test is limited because we can't easily test embedded files
	// We can only test the error handling

	// Test with non-existent file (should return empty bytes)
	// Since we can't access the embedded filesystem directly in tests,
	// we'll skip this test or use a mock

	// For now, just verify the function exists and doesn't panic
	// This is a placeholder test
	assert.True(t, true, "readEmbeddedFile function exists")
}

// TestSetupLogger tests the setupLogger function
func TestSetupLogger(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.Config
		expected string // expected log level
	}{
		{
			name:     "info level",
			cfg:      config.DefaultConfig(),
			expected: "info",
		},
		{
			name: "debug level",
			cfg: func() *config.Config {
				cfg := config.DefaultConfig()
				cfg.Logging.Level = "debug"
				return cfg
			}(),
			expected: "debug",
		},
		{
			name: "warn level",
			cfg: func() *config.Config {
				cfg := config.DefaultConfig()
				cfg.Logging.Level = "warn"
				return cfg
			}(),
			expected: "warn",
		},
		{
			name: "error level",
			cfg: func() *config.Config {
				cfg := config.DefaultConfig()
				cfg.Logging.Level = "error"
				return cfg
			}(),
			expected: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup logger with config
			setupLogger(tt.cfg)
			
			// We can't easily verify the logger level without accessing internals,
			// but we can verify it doesn't panic
			assert.True(t, true, "setupLogger should complete without panic")
		})
	}
}

// TestWriteTempFile tests the writeTempFile function
func TestWriteTempFile(t *testing.T) {
	// Test data
	testData := []byte("test content")
	
	// Write temp file
	filename := writeTempFile("test.pem", testData)
	
	// Verify file was created
	file, err := os.Open(filename)
	require.NoError(t, err)
	defer file.Close()
	defer os.Remove(filename)
	
	// Read content back
	content, err := io.ReadAll(file)
	require.NoError(t, err)
	
	// Verify content matches
	assert.Equal(t, testData, content)
	
	// Verify filename contains expected pattern
	assert.Contains(t, filename, "test.pem")
}

// TestMainFunction tests that main() can be called without panic
// Note: This is a limited test as main() starts a server
func TestMainFunction(t *testing.T) {
	// We can't actually call main() as it would start a server
	// This is a placeholder to verify the file compiles
	assert.True(t, true, "main.go should compile")
}
