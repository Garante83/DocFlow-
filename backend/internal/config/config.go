package config

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config enthaelt alle Anwendungskonfigurationen
type Config struct {
	Server struct {
		Port        string `mapstructure:"port" json:"port"`
		Host        string `mapstructure:"host" json:"host"`
		TLSCertPath string `mapstructure:"tls_cert_path" json:"tls_cert_path"`
		TLSKeyPath  string `mapstructure:"tls_key_path" json:"tls_key_path"`
	} `mapstructure:"server" json:"server"`

	Session struct {
		Timeout           time.Duration `mapstructure:"timeout" json:"timeout"`
		CleanupInterval   time.Duration `mapstructure:"cleanup_interval" json:"cleanup_interval"`
		MaxFailedAttempts int           `mapstructure:"max_failed_attempts" json:"max_failed_attempts"`
		LockoutDuration   time.Duration `mapstructure:"lockout_duration" json:"lockout_duration"`
	} `mapstructure:"session" json:"session"`

	Upload struct {
		MaxFileSizeMB int      `mapstructure:"max_file_size_mb" json:"max_file_size_mb"`
		AllowedTypes  []string `mapstructure:"allowed_types" json:"allowed_types"`
	} `mapstructure:"upload" json:"upload"`

	WebSocket struct {
		ReadDeadline    time.Duration `mapstructure:"read_deadline" json:"read_deadline"`
		PingInterval    time.Duration `mapstructure:"ping_interval" json:"ping_interval"`
		AllowedOrigins  []string      `mapstructure:"allowed_origins" json:"allowed_origins"`
		AllowPrivateIPs bool          `mapstructure:"allow_private_ips" json:"allow_private_ips"`
	} `mapstructure:"websocket" json:"websocket"`

	Logging struct {
		Level  string `mapstructure:"level" json:"level"`   // debug, info, warn, error
		Format string `mapstructure:"format" json:"format"` // json, text
	} `mapstructure:"logging" json:"logging"`
}

// DefaultConfig gibt Standardwerte zuruck
func DefaultConfig() *Config {
	cfg := &Config{}

	// Server
	cfg.Server.Port = "8082"
	cfg.Server.Host = "0.0.0.0"
	cfg.Server.TLSCertPath = ""
	cfg.Server.TLSKeyPath = ""

	// Session
	cfg.Session.Timeout = 1 * time.Hour
	cfg.Session.CleanupInterval = 5 * time.Minute
	cfg.Session.MaxFailedAttempts = 3
	cfg.Session.LockoutDuration = 5 * time.Minute

	// Upload
	cfg.Upload.MaxFileSizeMB = 10
	cfg.Upload.AllowedTypes = []string{"image/jpeg", "image/png", "image/webp"}

	// WebSocket
	cfg.WebSocket.ReadDeadline = 60 * time.Second
	cfg.WebSocket.PingInterval = 30 * time.Second
	cfg.WebSocket.AllowedOrigins = []string{"https://localhost:8082", "https://127.0.0.1:8082", "http://localhost:8082", "http://127.0.0.1:8082"}
	cfg.WebSocket.AllowPrivateIPs = true

	// Logging
	cfg.Logging.Level = "info"
	cfg.Logging.Format = "json"

	return cfg
}

// LoadConfig laedt die Konfiguration aus verschiedenen Quellen
func LoadConfig(configPath string) (*Config, error) {
	// 1. Erstelle Viper Instanz
	v := viper.New()
	v.SetConfigName("config") // Name der Config-Datei (ohne Endung)
	v.SetConfigType("yaml")   // oder json
	v.AutomaticEnv()          // Lese Umgebungsvariablen automatisch
	v.SetEnvPrefix("DSCAN")   // Praefix fuer Umgebungsvariablen (z.B. DSCAN_SERVER_PORT)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 2. Fuege Suchpfade hinzu
	if configPath != "" {
		v.AddConfigPath(configPath)
	}
	// Standardpfade
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/dokumentenscanner")

	// 3. Lese Config-Datei (falls existiert)
	if err := v.ReadInConfig(); err != nil {
		// Ignoriere "Config File Not Found"-Error, wir verwenden Defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// 4. Parse CLI Flags (ueberschreibt Config-Datei)
	// Nur wenn flag.Parsed() == false, d.h. flag.Parse() wurde noch nicht aufgerufen
	if !flag.Parsed() {
		parseFlags(v)
		flag.Parse()
	}

	// 5. Binde Umgebungsvariablen explizit in Viper
	// Dies ist notwendig, da Viper.AutomaticEnv() nur funktioniert,
	// wenn die Config-Datei die Struktur definiert. Ohne Config-Datei
	// mussen wir die Werte manuell ubertragen.
	bindEnvVars(v)

	// 6. Erstelle Config-Objekt mit Defaults
	cfg := DefaultConfig()

	// 7. Unmarshal in Struct (ueberschreibt Defaults mit Config-Datei/Umgebungsvariablen/Flags)
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 7. Validierung
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// parseFlags registriert CLI-Flags und setzt sie in Viper
// ACHTUNG: flag.Parse() wird NICHT hier aufgerufen, da dies zu Konflikten
// mit Test-Flags fuhren kann. Der Aufrufer muss flag.Parse() selbst aufrufen.
func parseFlags(v *viper.Viper) {
	flag.String("port", v.GetString("server.port"), "Server port")
	flag.String("host", v.GetString("server.host"), "Server host")
	flag.String("config", "", "Path to config file")

	// WebSocket spezifisch
	flag.Bool("ws-allow-private-ips", v.GetBool("websocket.allow_private_ips"), "Allow connections from private IPs")
	flag.String("ws-origins", strings.Join(v.GetStringSlice("websocket.allowed_origins"), ","), "Comma-separated allowed origins")

	// Setze Flag-Werte in Viper (nur wenn Flags gesetzt wurden)
	if f := flag.Lookup("port"); f != nil && f.Value.String() != f.DefValue {
		v.Set("server.port", f.Value.String())
	}
	if f := flag.Lookup("host"); f != nil && f.Value.String() != f.DefValue {
		v.Set("server.host", f.Value.String())
	}
	if f := flag.Lookup("ws-allow-private-ips"); f != nil && f.Value.String() != f.DefValue {
		v.Set("websocket.allow_private_ips", f.Value.String() == "true")
	}
	if f := flag.Lookup("ws-origins"); f != nil && f.Value.String() != f.DefValue {
		origins := strings.Split(f.Value.String(), ",")
		v.Set("websocket.allowed_origins", origins)
	}
}

// validateConfig validiert die geladene Konfiguration
func validateConfig(cfg *Config) error {
	if cfg.Server.Port == "" {
		return errors.New("server.port must be set")
	}

	if cfg.Upload.MaxFileSizeMB <= 0 {
		return errors.New("upload.max_file_size_mb must be > 0")
	}

	if cfg.Session.Timeout <= 0 {
		return errors.New("session.timeout must be > 0")
	}

	return nil
}

// bindEnvVars bindet Umgebungsvariablen explizit an Viper
// Dies ist notwendig, da Viper.AutomaticEnv() ohne Config-Datei nicht alle Variablen erkennt
func bindEnvVars(v *viper.Viper) {
	// Server
	if port := os.Getenv("DSCAN_SERVER_PORT"); port != "" {
		v.Set("server.port", port)
	}
	if host := os.Getenv("DSCAN_SERVER_HOST"); host != "" {
		v.Set("server.host", host)
	}
	if certPath := os.Getenv("DSCAN_SERVER_TLS_CERT_PATH"); certPath != "" {
		v.Set("server.tls_cert_path", certPath)
	}
	if keyPath := os.Getenv("DSCAN_SERVER_TLS_KEY_PATH"); keyPath != "" {
		v.Set("server.tls_key_path", keyPath)
	}

	// Session
	if timeout := os.Getenv("DSCAN_SESSION_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			v.Set("session.timeout", d)
		}
	}
	if cleanupInterval := os.Getenv("DSCAN_SESSION_CLEANUP_INTERVAL"); cleanupInterval != "" {
		if d, err := time.ParseDuration(cleanupInterval); err == nil {
			v.Set("session.cleanup_interval", d)
		}
	}
	if maxAttempts := os.Getenv("DSCAN_SESSION_MAX_FAILED_ATTEMPTS"); maxAttempts != "" {
		if i, err := strconv.Atoi(maxAttempts); err == nil {
			v.Set("session.max_failed_attempts", i)
		}
	}
	if lockoutDuration := os.Getenv("DSCAN_SESSION_LOCKOUT_DURATION"); lockoutDuration != "" {
		if d, err := time.ParseDuration(lockoutDuration); err == nil {
			v.Set("session.lockout_duration", d)
		}
	}

	// Upload
	if maxSize := os.Getenv("DSCAN_UPLOAD_MAX_FILE_SIZE_MB"); maxSize != "" {
		if i, err := strconv.Atoi(maxSize); err == nil {
			v.Set("upload.max_file_size_mb", i)
		}
	}

	// WebSocket
	if readDeadline := os.Getenv("DSCAN_WEB_SOCKET_READ_DEADLINE"); readDeadline != "" {
		if d, err := time.ParseDuration(readDeadline); err == nil {
			v.Set("websocket.read_deadline", d)
		}
	}
	if pingInterval := os.Getenv("DSCAN_WEB_SOCKET_PING_INTERVAL"); pingInterval != "" {
		if d, err := time.ParseDuration(pingInterval); err == nil {
			v.Set("websocket.ping_interval", d)
		}
	}
	if allowPrivateIPs := os.Getenv("DSCAN_WEB_SOCKET_ALLOW_PRIVATE_IPS"); allowPrivateIPs != "" {
		v.Set("websocket.allow_private_ips", allowPrivateIPs == "true")
	}

	// Logging
	if level := os.Getenv("DSCAN_LOGGING_LEVEL"); level != "" {
		v.Set("logging.level", level)
	}
	if format := os.Getenv("DSCAN_LOGGING_FORMAT"); format != "" {
		v.Set("logging.format", format)
	}
}

// ToJSON gibt die Konfiguration als JSON zurueck (fuer Debugging)
func (c *Config) ToJSON() (string, error) {
	bytes, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
