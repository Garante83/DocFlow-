package config

import (
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

//go:embed default_config.yaml
var defaultConfigYAML string

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

	PDF struct {
		MaxPages       int  `mapstructure:"max_pages" json:"max_pages"`
		JPEGQuality    int  `mapstructure:"jpeg_quality" json:"jpeg_quality"`
		CompressOutput bool `mapstructure:"compress_output" json:"compress_output"`
	} `mapstructure:"pdf" json:"pdf"`

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

	RateLimit struct {
		Enabled       bool `mapstructure:"enabled" json:"enabled"`
		MaxRequests   int  `mapstructure:"max_requests" json:"max_requests"`     // pro Fenster
		WindowSeconds int  `mapstructure:"window_seconds" json:"window_seconds"` // Fenster in Sekunden
	} `mapstructure:"rate_limit" json:"rate_limit"`
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

	// PDF
	cfg.PDF.MaxPages = 20
	cfg.PDF.JPEGQuality = 85
	cfg.PDF.CompressOutput = true

	// WebSocket
	cfg.WebSocket.ReadDeadline = 60 * time.Second
	cfg.WebSocket.PingInterval = 30 * time.Second
	cfg.WebSocket.AllowedOrigins = []string{"https://localhost:8082", "https://127.0.0.1:8082", "http://localhost:8082", "http://127.0.0.1:8082"}
	cfg.WebSocket.AllowPrivateIPs = true

	// Logging
	cfg.Logging.Level = "info"
	cfg.Logging.Format = "json"

	// Rate Limiting
	cfg.RateLimit.Enabled = true
	cfg.RateLimit.MaxRequests = 100
	cfg.RateLimit.WindowSeconds = 60

	return cfg
}

// LoadConfig laedt die Konfiguration aus verschiedenen Quellen
func LoadConfig(configPath string) (*Config, error) {
	// 0. Pre-Scan: --config Flag aus os.Args lesen, bevor irgendetwas anderes
	// passiert (die restlichen Flags brauchen Viper-Defaults als DefValue)
	if configPath == "" {
		configPath = preScanConfigPath(os.Args[1:])
	}

	// 1. Erstelle Viper Instanz
	v := viper.New()
	v.SetConfigName("config") // Name der Config-Datei (ohne Endung)
	v.SetConfigType("yaml")   // oder json
	v.AutomaticEnv()          // Lese Umgebungsvariablen automatisch
	v.SetEnvPrefix("DSCAN")   // Praefix fuer Umgebungsvariablen (z.B. DSCAN_SERVER_PORT)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 2. Suchpfade oder explizite Datei
	explicitFile := false
	if configPath != "" {
		v.SetConfigFile(configPath)
		explicitFile = true
	} else {
		// Standardpfade
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
		v.AddConfigPath("/etc/docflow")
	}

	// 3. Lese Config-Datei (falls existiert)
	if err := v.ReadInConfig(); err != nil {
		if explicitFile {
			// Explizite Datei: Jeder Fehler ist fatal (auch nicht gefunden)
			if isNotFoundError(err) {
				return nil, fmt.Errorf("config file not found: %s", configPath)
			}
			return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
		}
		// Suchpfade: "Nicht gefunden" wird ignoriert und die Default-Config
		// beim ersten Start an einen schreibbaren Ort geschrieben
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		writeDefaultConfig()
	}

	// 4. Parse CLI Flags (ueberschreibt Config-Datei)
	// Nur wenn flag.Parsed() == false, d.h. flag.Parse() wurde noch nicht aufgerufen
	if !flag.Parsed() {
		registerFlags(v)
		flag.Parse()
		applyFlagsToViper(v)
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

// preScanConfigPath sucht --config in den Argumenten, bevor die eigentliche
// Flag-Verarbeitung laeuft (diese braucht Viper-Defaults als Basis).
// Unterstuetzt "--config pfad" und "--config=pfad" (auch einfach-Bindestrich).
func preScanConfigPath(args []string) string {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "-config" || arg == "--config" {
			if i+1 < len(args) {
				return args[i+1]
			}
			return ""
		}
		if strings.HasPrefix(arg, "-config=") {
			return strings.TrimPrefix(arg, "-config=")
		}
		if strings.HasPrefix(arg, "--config=") {
			return strings.TrimPrefix(arg, "--config=")
		}
	}
	return ""
}

// isNotFoundError prueft ob der Viper-Fehler auf eine fehlende Datei hinweist
func isNotFoundError(err error) bool {
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		return os.IsNotExist(pathErr)
	}
	return false
}

// writeDefaultConfig schreibt die eingebettete Default-Config an den ersten
// schreibbaren Kandidatenpfad. Fehler werden nur geloggt, der Start laeuft
// mit Built-in-Defaults weiter (z.B. read-only Dateisysteme).
var writeDefaultConfigCandidates = []string{"config.yaml", "config/config.yaml", "/etc/docflow/config.yaml"}

func writeDefaultConfig() {
	for _, path := range writeDefaultConfigCandidates {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			continue
		}
		if err := os.WriteFile(path, []byte(defaultConfigYAML), 0o644); err != nil {
			continue
		}
		slog.Info("Wrote default config file", "path", path)
		return
	}
	slog.Warn("No writable location for default config, using built-in defaults")
}

// registerFlags registriert CLI-Flags mit Viper-Defaults als DefValue.
// ACHTUNG: flag.Parse() wird NICHT hier aufgerufen, da dies zu Konflikten
// mit Test-Flags fuehren kann. Der Aufrufer muss flag.Parse() selbst aufrufen.
func registerFlags(v *viper.Viper) {
	flag.String("port", v.GetString("server.port"), "Server port")
	flag.String("host", v.GetString("server.host"), "Server host")
	flag.String("config", "", "Path to config file")

	// WebSocket spezifisch
	flag.Bool("ws-allow-private-ips", v.GetBool("websocket.allow_private_ips"), "Allow connections from private IPs")
	flag.String("ws-origins", strings.Join(v.GetStringSlice("websocket.allowed_origins"), ","), "Comma-separated allowed origins")
}

// applyFlagsToViper uebertraegt geparste Flag-Werte in Viper. Muss NACH
// flag.Parse() laufen, da vorher Value == DefValue gilt.
func applyFlagsToViper(v *viper.Viper) {
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
	if allowedTypes := os.Getenv("DSCAN_UPLOAD_ALLOWED_TYPES"); allowedTypes != "" {
		v.Set("upload.allowed_types", strings.Split(allowedTypes, ","))
	}

	// PDF
	if maxPages := os.Getenv("DSCAN_PDF_MAX_PAGES"); maxPages != "" {
		if i, err := strconv.Atoi(maxPages); err == nil {
			v.Set("pdf.max_pages", i)
		}
	}
	if quality := os.Getenv("DSCAN_PDF_JPEG_QUALITY"); quality != "" {
		if i, err := strconv.Atoi(quality); err == nil {
			v.Set("pdf.jpeg_quality", i)
		}
	}
	if compress := os.Getenv("DSCAN_PDF_COMPRESS_OUTPUT"); compress != "" {
		v.Set("pdf.compress_output", compress == "true")
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
	if origins := os.Getenv("DSCAN_WEB_SOCKET_ALLOWED_ORIGINS"); origins != "" {
		v.Set("websocket.allowed_origins", strings.Split(origins, ","))
	}

	// Logging
	if level := os.Getenv("DSCAN_LOGGING_LEVEL"); level != "" {
		v.Set("logging.level", level)
	}
	if format := os.Getenv("DSCAN_LOGGING_FORMAT"); format != "" {
		v.Set("logging.format", format)
	}

	// Rate Limiting
	if enabled := os.Getenv("DSCAN_RATE_LIMIT_ENABLED"); enabled != "" {
		v.Set("rate_limit.enabled", enabled == "true")
	}
	if maxRequests := os.Getenv("DSCAN_RATE_LIMIT_MAX_REQUESTS"); maxRequests != "" {
		if i, err := strconv.Atoi(maxRequests); err == nil {
			v.Set("rate_limit.max_requests", i)
		}
	}
	if windowSeconds := os.Getenv("DSCAN_RATE_LIMIT_WINDOW_SECONDS"); windowSeconds != "" {
		if i, err := strconv.Atoi(windowSeconds); err == nil {
			v.Set("rate_limit.window_seconds", i)
		}
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
