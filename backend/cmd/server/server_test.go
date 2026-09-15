package main

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"docflow/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// === generateSelfSignedCert Tests ===

func TestGenerateSelfSignedCert(t *testing.T) {
	certPath, keyPath := generateSelfSignedCert()
	defer os.Remove(certPath)
	defer os.Remove(keyPath)

	require.NotEmpty(t, certPath)
	require.NotEmpty(t, keyPath)

	certPEM, err := os.ReadFile(certPath)
	require.NoError(t, err)
	keyPEM, err := os.ReadFile(keyPath)
	require.NoError(t, err)

	// Certificate must be a valid PEM block with a parseable x509 cert
	certBlock, _ := pem.Decode(certPEM)
	require.NotNil(t, certBlock, "cert must be PEM encoded")
	assert.Equal(t, "CERTIFICATE", certBlock.Type)

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	require.NoError(t, err)
	assert.Equal(t, []string{"DocFlow"}, cert.Subject.Organization)
	assert.WithinDuration(t, time.Now(), cert.NotBefore, time.Minute)
	assert.WithinDuration(t, time.Now().Add(365*24*time.Hour), cert.NotAfter, time.Minute)
	assert.Contains(t, cert.ExtKeyUsage, x509.ExtKeyUsageServerAuth)

	// Key must be a valid EC private key
	keyBlock, _ := pem.Decode(keyPEM)
	require.NotNil(t, keyBlock, "key must be PEM encoded")
	assert.Equal(t, "EC PRIVATE KEY", keyBlock.Type)

	key, err := x509.ParseECPrivateKey(keyBlock.Bytes)
	require.NoError(t, err)
	assert.IsType(t, &ecdsa.PrivateKey{}, key)

	// Cert and key must form a matching pair
	pair, err := tls.LoadX509KeyPair(certPath, keyPath)
	require.NoError(t, err, "cert and key must form a valid TLS pair")
	pairLeaf, ok := pair.Leaf.PublicKey.(*ecdsa.PublicKey)
	require.True(t, ok, "leaf public key must be an ECDSA key")
	assert.Equal(t, key.PublicKey, *pairLeaf)
}

// === rateLimiter cleanup/Stop Tests ===

func newTestRateLimiter() *rateLimiter {
	return &rateLimiter{
		clients:  make(map[string]*clientInfo),
		enabled:  true,
		maxReqs:  5,
		window:   time.Minute,
		stopChan: make(chan struct{}),
	}
}

func TestRateLimiterCleanup_RemovesStaleClients(t *testing.T) {
	rl := newTestRateLimiter()
	defer rl.Stop()

	// Stale client: last access 20 min ago, no in-flight requests
	rl.mu.Lock()
	rl.clients["9.9.9.9"] = &clientInfo{lastAccess: time.Now().Add(-20 * time.Minute)}
	// Recent client must be kept
	rl.clients["1.1.1.1"] = &clientInfo{lastAccess: time.Now(), requests: []time.Time{time.Now()}}
	// Stale but with requests must be kept (requests still counted)
	rl.clients["2.2.2.2"] = &clientInfo{lastAccess: time.Now().Add(-20 * time.Minute), requests: []time.Time{time.Now().Add(-15 * time.Minute)}}
	rl.mu.Unlock()

	go rl.cleanup(10 * time.Millisecond)

	assert.Eventually(t, func() bool {
		rl.mu.Lock()
		defer rl.mu.Unlock()
		_, staleGone := rl.clients["9.9.9.9"]
		_, recent := rl.clients["1.1.1.1"]
		_, withRequests := rl.clients["2.2.2.2"]
		return !staleGone && recent && withRequests
	}, 2*time.Second, 10*time.Millisecond, "only the stale request-free client must be removed")
}

func TestRateLimiterCleanup_StopTerminates(t *testing.T) {
	rl := newTestRateLimiter()

	done := make(chan struct{})
	go func() {
		rl.cleanup(10 * time.Millisecond)
		close(done)
	}()

	rl.Stop()

	select {
	case <-done:
		// goroutine terminated
	case <-time.After(2 * time.Second):
		t.Error("cleanup goroutine did not stop after Stop()")
	}
}

// === rateLimitMiddleware Tests ===

func TestRateLimitMiddleware_BlocksAfterMaxRequests(t *testing.T) {
	rl := newTestRateLimiter()
	defer rl.Stop()
	rl.maxReqs = 2

	r := gin.New()
	r.Use(rateLimitMiddleware(rl))
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	// First two requests pass
	for i := 0; i < 2; i++ {
		w := performRequest(r, "GET", "/test")
		assert.Equal(t, http.StatusOK, w.Code, "request %d must pass", i+1)
	}

	// Third request is limited
	w := performRequest(r, "GET", "/test")
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "60", w.Header().Get("Retry-After"))
	assert.Contains(t, w.Body.String(), "rate limit exceeded")
}

func TestRateLimitMiddleware_DisabledLimiterAllowsAll(t *testing.T) {
	rl := newTestRateLimiter()
	defer rl.Stop()
	rl.enabled = false

	r := gin.New()
	r.Use(rateLimitMiddleware(rl))
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	for i := 0; i < 10; i++ {
		w := performRequest(r, "GET", "/test")
		assert.Equal(t, http.StatusOK, w.Code)
	}
}

func TestRateLimitMiddleware_WindowRecovery(t *testing.T) {
	rl := newTestRateLimiter()
	defer rl.Stop()
	rl.maxReqs = 1
	rl.window = 50 * time.Millisecond

	r := gin.New()
	r.Use(rateLimitMiddleware(rl))
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	assert.Equal(t, http.StatusOK, performRequest(r, "GET", "/test").Code)
	assert.Equal(t, http.StatusTooManyRequests, performRequest(r, "GET", "/test").Code)

	// After the window, the request slot is free again
	time.Sleep(60 * time.Millisecond)
	assert.Equal(t, http.StatusOK, performRequest(r, "GET", "/test").Code)
}

// === privacyLogFormatter Tests ===

func TestPrivacyLogFormatter_QueryStringStripped(t *testing.T) {
	out := privacyLogFormatter(gin.LogFormatterParams{
		Method:     "GET",
		Path:       "/api/session?token=secret",
		StatusCode: 200,
		Latency:    5 * time.Millisecond,
	})

	assert.NotContains(t, out, "token", "query string must never appear in access logs")
	assert.NotContains(t, out, "secret")
	assert.Contains(t, out, "GET /api/session 200")
}

func TestPrivacyLogFormatter_NoQueryString(t *testing.T) {
	out := privacyLogFormatter(gin.LogFormatterParams{
		Method:     "POST",
		Path:       "/api/session",
		StatusCode: 201,
		Latency:    time.Millisecond,
	})

	assert.Contains(t, out, "POST /api/session 201")
}

// === runServer Tests ===

func TestRunServer_StartsServesAndShutsDown(t *testing.T) {
	// Pre-bind a listener on a random port and hand it to runServer
	ln, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer ln.Close()

	cfg := config.DefaultConfig()
	quit := make(chan os.Signal, 1)

	errCh := make(chan error, 1)
	go func() {
		errCh <- runServer(cfg, quit, ln)
	}()

	// Wait for the server to accept TLS connections
	addr := ln.Addr().String()
	var resp *http.Response
	require.Eventually(t, func() bool {
		client := &http.Client{
			Timeout: 1 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // self-signed test cert
			},
		}
		r, err := client.Get(fmt.Sprintf("https://%s/", addr))
		if err != nil {
			return false
		}
		resp = r
		return true
	}, 5*time.Second, 50*time.Millisecond, "server must accept HTTPS requests")
	defer resp.Body.Close()

	// Shut down via the quit channel and expect a clean return
	quit <- os.Interrupt
	select {
	case err := <-errCh:
		assert.NoError(t, err, "runServer must return nil after graceful shutdown")
	case <-time.After(15 * time.Second):
		t.Error("runServer did not return after shutdown signal")
	}
}

func TestRunServer_ListenerError(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Server.Port = "invalid-port" // net.Listen("tcp", ":invalid-port") fails

	quit := make(chan os.Signal, 1)
	err := runServer(cfg, quit, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to listen")
}

// helper: request via gin test recorder
func performRequest(r *gin.Engine, method, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, nil)
	r.ServeHTTP(w, req)
	return w
}
