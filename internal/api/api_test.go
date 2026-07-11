package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/craftlabs/video-stream-capture-engine/internal/auth"
	"github.com/craftlabs/video-stream-capture-engine/internal/config"
	"github.com/craftlabs/video-stream-capture-engine/internal/manager"
	"github.com/craftlabs/video-stream-capture-engine/internal/store"
)

type mockStore struct {
	users  map[string]*store.User
	events []store.Event
	nextID int
}

func newMockStore() *mockStore {
	return &mockStore{
		users:  make(map[string]*store.User),
		events: make([]store.Event, 0),
		nextID: 1,
	}
}

func (m *mockStore) GetUserByUsername(_ context.Context, username string) (*store.User, error) {
	u, ok := m.users[username]
	if !ok {
		return nil, &mockErr{"user not found"}
	}
	return u, nil
}

func (m *mockStore) CreateUser(_ context.Context, username, passwordHash string) error {
	m.users[username] = &store.User{ID: len(m.users) + 1, Username: username, PasswordHash: passwordHash}
	return nil
}

func (m *mockStore) UpdatePassword(_ context.Context, username, newHash string) error {
	u, ok := m.users[username]
	if !ok {
		return &mockErr{"user not found"}
	}
	u.PasswordHash = newHash
	return nil
}

func (m *mockStore) InsertEvent(_ context.Context, streamID, level, message string) error {
	ID := m.nextID
	m.nextID++
	m.events = append(m.events, store.Event{
		ID:       ID,
		StreamID: streamID,
		Level:    level,
		Message:  message,
	})
	return nil
}

func (m *mockStore) ListEvents(_ context.Context, f store.EventFilter) ([]store.Event, int, error) {
	return m.events, len(m.events), nil
}

func (m *mockStore) AckEvents(_ context.Context, ids []int) error {
	return nil
}

func (m *mockStore) AckAllEvents(_ context.Context) error {
	return nil
}

func (m *mockStore) Close() {}
func (m *mockStore) Migrate(_ context.Context) error { return nil }

type mockErr struct{ msg string }

func (e *mockErr) Error() string { return e.msg }

func newTestServer(t *testing.T) *Server {
	t.Helper()
	st := newMockStore()
	adminHash, err := auth.HashPassword("admin")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if err := st.CreateUser(context.Background(), "admin", adminHash); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	return &Server{
		cfg:    &config.Config{},
		mgr:    &manager.StreamManager{},
		db:     st,
		jwtKey: []byte("test-jwt-secret"),
	}
}

func TestHandleLogin_Success(t *testing.T) {
	s := newTestServer(t)

	body := `{"username":"admin","password":"admin"}`
	req := httptest.NewRequest("POST", "/api/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleLogin(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var resp struct {
		Code    int                    `json:"code"`
		Message string                 `json:"message"`
		Data    map[string]interface{} `json:"data"`
	}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Code != 0 {
		t.Errorf("code = %d, want 0", resp.Code)
	}
	if resp.Data == nil {
		t.Fatal("data is nil")
	}
	token, ok := resp.Data["token"]
	if !ok || token == "" {
		t.Error("token missing or empty")
	}
}

func TestHandleLogin_WrongPassword(t *testing.T) {
	s := newTestServer(t)

	body := `{"username":"admin","password":"wrongpass"}`
	req := httptest.NewRequest("POST", "/api/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleLogin(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestHandleLogin_UserNotFound(t *testing.T) {
	s := newTestServer(t)

	body := `{"username":"nonexistent","password":"pass"}`
	req := httptest.NewRequest("POST", "/api/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.handleLogin(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestHandleLogin_InvalidMethod(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest("GET", "/api/login", nil)
	w := httptest.NewRecorder()

	s.handleLogin(w, req)

	if w.Code != 405 {
		t.Errorf("status = %d, want 405", w.Code)
	}
}

func TestAuthMiddleware_NoToken(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest("GET", "/api/streams", nil)
	w := httptest.NewRecorder()

	s.authWrap(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called without token")
	}).ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest("GET", "/api/streams", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	s.authWrap(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with invalid token")
	}).ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestHandleRefresh_Success(t *testing.T) {
	s := newTestServer(t)

	body := `{}`
	req := httptest.NewRequest("POST", "/api/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	token, _ := auth.GenerateToken("admin", s.jwtKey, 24*time.Hour)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	s.authWrap(s.handleRefresh).ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestHandleMetricsSummary(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest("GET", "/api/metrics/summary", nil)
	req.Header.Set("Content-Type", "application/json")
	token, _ := auth.GenerateToken("admin", s.jwtKey, 24*time.Hour)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	s.authWrap(s.handleMetricsSummary).ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
}
