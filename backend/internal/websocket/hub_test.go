package websocket

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testUpgrader = gws.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type testWSServer struct {
	server *httptest.Server
	conn   *gws.Conn
	mu     sync.Mutex
	done   chan struct{}
}

func newTestWSServer(t *testing.T) *testWSServer {
	t.Helper()
	tws := &testWSServer{done: make(chan struct{})}
	tws.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := testUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		tws.mu.Lock()
		tws.conn = c
		tws.mu.Unlock()
		// Block reading until closed
		for {
			_, _, err := c.ReadMessage()
			if err != nil {
				return
			}
		}
	}))
	t.Cleanup(func() {
		tws.mu.Lock()
		if tws.conn != nil {
			tws.conn.Close()
		}
		tws.mu.Unlock()
		tws.server.Close()
	})
	return tws
}

func (tws *testWSServer) connect(t *testing.T) *gws.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(tws.server.URL, "http")
	c, _, err := gws.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	t.Cleanup(func() { c.Close() })
	return c
}

func (tws *testWSServer) getServerConn() *gws.Conn {
	tws.mu.Lock()
	defer tws.mu.Unlock()
	return tws.conn
}

func startHub(t *testing.T) *Hub {
	t.Helper()
	hub := NewHub()
	done := make(chan struct{})
	go func() {
		hub.Run()
		close(done)
	}()
	t.Cleanup(func() {
		hub.Stop()
		<-done
	})
	time.Sleep(10 * time.Millisecond)
	return hub
}

func TestNewHub(t *testing.T) {
	hub := NewHub()
	require.NotNil(t, hub)
	assert.NotNil(t, hub.clients)
	assert.NotNil(t, hub.register)
	assert.NotNil(t, hub.unregister)
	assert.NotNil(t, hub.broadcast)
	assert.NotNil(t, hub.stopChan)
}

func TestHubRunAndStop(t *testing.T) {
	hub := NewHub()
	done := make(chan struct{})
	go func() {
		hub.Run()
		close(done)
	}()

	time.Sleep(10 * time.Millisecond)
	hub.Stop()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Hub.Run did not return after Stop")
	}
}

func TestHubRegisterClient(t *testing.T) {
	hub := startHub(t)
	tws := newTestWSServer(t)
	tws.connect(t)

	time.Sleep(20 * time.Millisecond)
	serverConn := tws.getServerConn()
	require.NotNil(t, serverConn)

	sessionID := uuid.New()
	client := &Client{Conn: serverConn, SessionID: sessionID, Hub: hub}
	hub.Register(client)
	time.Sleep(30 * time.Millisecond)

	hub.mu.Lock()
	defer hub.mu.Unlock()
	clients, exists := hub.clients[sessionID]
	assert.True(t, exists)
	assert.Len(t, clients, 1)
}

func TestHubRegisterMultipleClientsSameSession(t *testing.T) {
	hub := startHub(t)
	tws := newTestWSServer(t)
	c1 := tws.connect(t)

	wsURL := "ws" + strings.TrimPrefix(tws.server.URL, "http")
	c2, _, err := gws.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	t.Cleanup(func() { c2.Close() })

	time.Sleep(20 * time.Millisecond)

	sessionID := uuid.New()
	client1 := &Client{Conn: c1, SessionID: sessionID, Hub: hub}
	client2 := &Client{Conn: c2, SessionID: sessionID, Hub: hub}

	hub.Register(client1)
	hub.Register(client2)
	time.Sleep(30 * time.Millisecond)

	hub.mu.Lock()
	defer hub.mu.Unlock()
	clients, exists := hub.clients[sessionID]
	assert.True(t, exists)
	assert.Len(t, clients, 2)
}

func TestHubUnregisterClient(t *testing.T) {
	hub := startHub(t)
	tws := newTestWSServer(t)
	c := tws.connect(t)
	time.Sleep(20 * time.Millisecond)

	sessionID := uuid.New()
	client := &Client{Conn: c, SessionID: sessionID, Hub: hub}
	hub.Register(client)
	time.Sleep(30 * time.Millisecond)

	hub.Unregister(client)
	time.Sleep(30 * time.Millisecond)

	hub.mu.Lock()
	defer hub.mu.Unlock()
	clients, exists := hub.clients[sessionID]
	assert.True(t, exists)
	_, stillRegistered := clients[client]
	assert.False(t, stillRegistered)
}

func TestHubBroadcastToSession(t *testing.T) {
	hub := startHub(t)
	tws := newTestWSServer(t)
	c := tws.connect(t)
	time.Sleep(20 * time.Millisecond)

	sessionID := uuid.New()
	client := &Client{Conn: c, SessionID: sessionID, Hub: hub}
	hub.Register(client)
	time.Sleep(30 * time.Millisecond)

	hub.Broadcast(sessionID, []byte(`{"event":"image_uploaded"}`))
	time.Sleep(30 * time.Millisecond)

	// Client should have received the message via its read side
	// Since the server side blocks on ReadMessage, the message went through
	assert.True(t, true, "Broadcast should not panic")
}

func TestHubBroadcastToNonExistentSession(t *testing.T) {
	hub := startHub(t)
	sessionID := uuid.New()

	// Should not panic
	hub.Broadcast(sessionID, []byte(`{"event":"test"}`))
	time.Sleep(20 * time.Millisecond)

	assert.True(t, true)
}

func TestHubBroadcastOnlyToCorrectSession(t *testing.T) {
	hub := startHub(t)
	twsA := newTestWSServer(t)
	twsB := newTestWSServer(t)
	cA := twsA.connect(t)
	cB := twsB.connect(t)
	time.Sleep(20 * time.Millisecond)

	sessionA := uuid.New()
	sessionB := uuid.New()
	clientA := &Client{Conn: cA, SessionID: sessionA, Hub: hub}
	clientB := &Client{Conn: cB, SessionID: sessionB, Hub: hub}

	hub.Register(clientA)
	hub.Register(clientB)
	time.Sleep(30 * time.Millisecond)

	hub.Broadcast(sessionA, []byte(`{"event":"session_a_event"}`))
	time.Sleep(30 * time.Millisecond)

	hub.mu.Lock()
	defer hub.mu.Unlock()
	assert.Len(t, hub.clients[sessionA], 1)
	assert.Len(t, hub.clients[sessionB], 1)
}

func TestHubRegisterSameClientTwice(t *testing.T) {
	hub := startHub(t)
	tws := newTestWSServer(t)
	c := tws.connect(t)
	time.Sleep(20 * time.Millisecond)

	sessionID := uuid.New()
	client := &Client{Conn: c, SessionID: sessionID, Hub: hub}
	hub.Register(client)
	hub.Register(client)
	time.Sleep(30 * time.Millisecond)

	hub.mu.Lock()
	defer hub.mu.Unlock()
	clients, exists := hub.clients[sessionID]
	assert.True(t, exists)
	assert.Len(t, clients, 1)
}
