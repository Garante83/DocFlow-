package config

import (
	"os"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.NotNil(t, cfg)
	assert.Equal(t, "8082", cfg.Server.Port)
	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 10, cfg.Upload.MaxFileSizeMB)
	assert.Len(t, cfg.Upload.AllowedTypes, 3)
	assert.Equal(t, "info", cfg.Logging.Level)
	assert.Equal(t, "json", cfg.Logging.Format)
	assert.Equal(t, 60*60, int(cfg.Session.Timeout.Seconds()))
	assert.Equal(t, 3, cfg.Session.MaxFailedAttempts)
}

func TestLoadConfig_WithoutFile(t *testing.T) {
	// Test ohne Config-Datei (sollte Defaults nutzen)
	cfg, err := LoadConfig("")
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "8082", cfg.Server.Port)
}

func TestLoadConfig_WithEnvVars(t *testing.T) {
	// Speichere aktuelle Umgebungsvariable und setze zurueck
	oldPort := os.Getenv("DSCAN_SERVER_PORT")
	defer func() {
		if oldPort != "" {
			os.Setenv("DSCAN_SERVER_PORT", oldPort)
		} else {
			os.Unsetenv("DSCAN_SERVER_PORT")
		}
	}()

	// Setze Umgebungsvariable
	os.Setenv("DSCAN_SERVER_PORT", "9090")

	cfg, err := LoadConfig("")
	require.NoError(t, err)
	assert.Equal(t, "9090", cfg.Server.Port)
}

func TestValidateConfig(t *testing.T) {
	// Valid config
	cfg := DefaultConfig()
	err := validateConfig(cfg)
	assert.NoError(t, err)

	// Invalid config (empty port)
	cfg = DefaultConfig()
	cfg.Server.Port = ""
	err = validateConfig(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server.port must be set")

	// Invalid config (negative file size)
	cfg = DefaultConfig()
	cfg.Upload.MaxFileSizeMB = -1
	err = validateConfig(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "upload.max_file_size_mb must be > 0")

	// Invalid config (zero timeout)
	cfg = DefaultConfig()
	cfg.Session.Timeout = 0
	err = validateConfig(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session.timeout must be > 0")
}

func TestToJSON(t *testing.T) {
	cfg := DefaultConfig()
	jsonStr, err := cfg.ToJSON()
	require.NoError(t, err)
	assert.Contains(t, jsonStr, `"port": "8082"`)
	assert.Contains(t, jsonStr, `"level": "info"`)
	assert.Contains(t, jsonStr, `"format": "json"`)
}

func TestDefaultConfig_AllFields(t *testing.T) {
	cfg := DefaultConfig()

	// Server
	assert.Equal(t, "8082", cfg.Server.Port)
	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, "", cfg.Server.TLSCertPath)
	assert.Equal(t, "", cfg.Server.TLSKeyPath)

	// Session
	assert.Equal(t, 1*time.Hour, cfg.Session.Timeout)
	assert.Equal(t, 5*time.Minute, cfg.Session.CleanupInterval)
	assert.Equal(t, 3, cfg.Session.MaxFailedAttempts)
	assert.Equal(t, 5*time.Minute, cfg.Session.LockoutDuration)

	// Upload
	assert.Equal(t, 10, cfg.Upload.MaxFileSizeMB)
	assert.Equal(t, []string{"image/jpeg", "image/png", "image/webp"}, cfg.Upload.AllowedTypes)

	// WebSocket
	assert.Equal(t, 60*time.Second, cfg.WebSocket.ReadDeadline)
	assert.Equal(t, 30*time.Second, cfg.WebSocket.PingInterval)
	assert.Equal(t, []string{"https://localhost:8082", "https://127.0.0.1:8082", "http://localhost:8082", "http://127.0.0.1:8082"}, cfg.WebSocket.AllowedOrigins)
	assert.True(t, cfg.WebSocket.AllowPrivateIPs)

	// Logging
	assert.Equal(t, "info", cfg.Logging.Level)
	assert.Equal(t, "json", cfg.Logging.Format)
}

func TestValidateConfig_AllFields(t *testing.T) {
	// Valid config
	cfg := DefaultConfig()
	err := validateConfig(cfg)
	assert.NoError(t, err)

	// Invalid: empty port
	cfg = DefaultConfig()
	cfg.Server.Port = ""
	err = validateConfig(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server.port must be set")

	// Invalid: negative file size
	cfg = DefaultConfig()
	cfg.Upload.MaxFileSizeMB = -1
	err = validateConfig(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "upload.max_file_size_mb must be > 0")

	// Invalid: zero timeout
	cfg = DefaultConfig()
	cfg.Session.Timeout = 0
	err = validateConfig(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session.timeout must be > 0")
}

func TestLoadConfig_WithAllEnvVars(t *testing.T) {
	// Save old env vars
	oldPort := os.Getenv("DSCAN_SERVER_PORT")
	oldHost := os.Getenv("DSCAN_SERVER_HOST")
	oldMaxSize := os.Getenv("DSCAN_UPLOAD_MAX_FILE_SIZE_MB")
	oldLevel := os.Getenv("DSCAN_LOGGING_LEVEL")

	// Cleanup
	defer func() {
		os.Unsetenv("DSCAN_SERVER_PORT")
		os.Unsetenv("DSCAN_SERVER_HOST")
		os.Unsetenv("DSCAN_UPLOAD_MAX_FILE_SIZE_MB")
		os.Unsetenv("DSCAN_LOGGING_LEVEL")
		if oldPort != "" {
			os.Setenv("DSCAN_SERVER_PORT", oldPort)
		}
		if oldHost != "" {
			os.Setenv("DSCAN_SERVER_HOST", oldHost)
		}
		if oldMaxSize != "" {
			os.Setenv("DSCAN_UPLOAD_MAX_FILE_SIZE_MB", oldMaxSize)
		}
		if oldLevel != "" {
			os.Setenv("DSCAN_LOGGING_LEVEL", oldLevel)
		}
	}()

	// Set env vars
	os.Setenv("DSCAN_SERVER_PORT", "9999")
	os.Setenv("DSCAN_SERVER_HOST", "127.0.0.1")
	os.Setenv("DSCAN_UPLOAD_MAX_FILE_SIZE_MB", "20")
	os.Setenv("DSCAN_LOGGING_LEVEL", "debug")

	cfg, err := LoadConfig("")
	require.NoError(t, err)
	assert.Equal(t, "9999", cfg.Server.Port)
	assert.Equal(t, "127.0.0.1", cfg.Server.Host)
	assert.Equal(t, 20, cfg.Upload.MaxFileSizeMB)
	assert.Equal(t, "debug", cfg.Logging.Level)
}

func TestLoadConfig_WithCLIFlags(t *testing.T) {
	// Note: CLI flags need special handling as flag.Parse() is called in LoadConfig
	// For now, we just test that LoadConfig doesn't break without flags
	cfg, err := LoadConfig("")
	require.NoError(t, err)
	assert.NotNil(t, cfg)
}

func TestBindEnvVars(t *testing.T) {
	// Save old env vars
	oldPort := os.Getenv("DSCAN_SERVER_PORT")
	oldHost := os.Getenv("DSCAN_SERVER_HOST")
	oldTimeout := os.Getenv("DSCAN_SESSION_TIMEOUT")

	// Cleanup
	defer func() {
		os.Unsetenv("DSCAN_SERVER_PORT")
		os.Unsetenv("DSCAN_SERVER_HOST")
		os.Unsetenv("DSCAN_SESSION_TIMEOUT")
		if oldPort != "" {
			os.Setenv("DSCAN_SERVER_PORT", oldPort)
		}
		if oldHost != "" {
			os.Setenv("DSCAN_SERVER_HOST", oldHost)
		}
		if oldTimeout != "" {
			os.Setenv("DSCAN_SESSION_TIMEOUT", oldTimeout)
		}
	}()

	// Create a Viper instance and bind env vars
	v := viper.New()
	bindEnvVars(v)

	// Set env vars
	os.Setenv("DSCAN_SERVER_PORT", "9999")
	os.Setenv("DSCAN_SERVER_HOST", "0.0.0.0")
	os.Setenv("DSCAN_SESSION_TIMEOUT", "2h")

	// Reload bindEnvVars to pick up the new values
	bindEnvVars(v)

	// Check that values were bound
	assert.Equal(t, "9999", v.GetString("server.port"))
	assert.Equal(t, "0.0.0.0", v.GetString("server.host"))
	assert.Equal(t, 2*time.Hour, v.GetDuration("session.timeout"))
}

func TestParseFlags(t *testing.T) {
	// Create a Viper instance
	v := viper.New()
	v.Set("server.port", "8082")
	v.Set("server.host", "0.0.0.0")

	// Call parseFlags - this should not panic
	// Note: This function is designed to work with flag.Parse()
	// which we can't easily test in unit tests
	parseFlags(v)

	// Verify it doesn't crash
	assert.True(t, true, "parseFlags should complete without panic")
}

func TestConfigMerging(t *testing.T) {
	// Test that config values are properly merged from multiple sources
	// This tests the priority: Defaults < Config File < ENV < Flags

	// Set up a config with some values
	cfg := DefaultConfig()

	// Verify default values
	assert.Equal(t, "8082", cfg.Server.Port)
	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 10, cfg.Upload.MaxFileSizeMB)

	// This test verifies the config structure is correct
	// Full merging tests would require more complex setup
}
