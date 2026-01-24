// Package serve provides an HTTP server for NTM with REST API and event streaming.
package serve

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Dicklesworthstone/ntm/internal/events"
	"github.com/Dicklesworthstone/ntm/internal/state"
)

// Server provides HTTP API and event streaming for NTM.
type Server struct {
	host       string
	port       int
	eventBus   *events.EventBus
	stateStore *state.Store
	server     *http.Server
	auth       AuthConfig

	// SSE clients
	sseClients   map[chan events.BusEvent]struct{}
	sseClientsMu sync.RWMutex

	corsAllowedOrigins []string
	jwksCache          *jwksCache
}

// AuthMode configures authentication for the server.
type AuthMode string

const (
	AuthModeLocal  AuthMode = "local"
	AuthModeAPIKey AuthMode = "api_key"
	AuthModeOIDC   AuthMode = "oidc"
	AuthModeMTLS   AuthMode = "mtls"
)

// AuthConfig holds server authentication configuration.
type AuthConfig struct {
	Mode   AuthMode
	APIKey string
	OIDC   OIDCConfig
	MTLS   MTLSConfig
}

// OIDCConfig configures OIDC/JWT verification for API access.
type OIDCConfig struct {
	Issuer   string
	Audience string
	JWKSURL  string
	CacheTTL time.Duration
}

// MTLSConfig configures mutual TLS for API access.
type MTLSConfig struct {
	CertFile     string
	KeyFile      string
	ClientCAFile string
}

// Config holds server configuration.
type Config struct {
	Host       string
	Port       int
	EventBus   *events.EventBus
	StateStore *state.Store
	Auth       AuthConfig
	// AllowedOrigins controls CORS origin allowlist. Empty means default localhost only.
	AllowedOrigins []string
}

const (
	defaultPort         = 7337
	defaultJWKSCacheTTL = 10 * time.Minute
)

const requestIDHeader = "X-Request-Id"

type ctxKey string

const requestIDKey ctxKey = "request_id"

func ParseAuthMode(raw string) (AuthMode, error) {
	mode := AuthMode(strings.ToLower(strings.TrimSpace(raw)))
	switch mode {
	case "", AuthModeLocal:
		return AuthModeLocal, nil
	case AuthModeAPIKey, AuthModeOIDC, AuthModeMTLS:
		return mode, nil
	default:
		return "", fmt.Errorf("invalid auth mode %q (valid: local, api_key, oidc, mtls)", raw)
	}
}

func defaultLocalOrigins() []string {
	return []string{
		"http://localhost",
		"http://127.0.0.1",
		"http://[::1]",
		"https://localhost",
		"https://127.0.0.1",
		"https://[::1]",
	}
}

func applyDefaults(cfg *Config) {
	if cfg.Port == 0 {
		cfg.Port = defaultPort
	}
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	if cfg.Auth.Mode == "" {
		cfg.Auth.Mode = AuthModeLocal
	}
	if cfg.Auth.OIDC.CacheTTL == 0 {
		cfg.Auth.OIDC.CacheTTL = defaultJWKSCacheTTL
	}
	if len(cfg.AllowedOrigins) == 0 {
		cfg.AllowedOrigins = defaultLocalOrigins()
	}
}

// ValidateConfig checks server configuration for security and completeness.
func ValidateConfig(cfg Config) error {
	applyDefaults(&cfg)

	mode, err := ParseAuthMode(string(cfg.Auth.Mode))
	if err != nil {
		return err
	}
	cfg.Auth.Mode = mode

	if mode == AuthModeAPIKey && cfg.Auth.APIKey == "" {
		return fmt.Errorf("auth mode api_key requires --api-key")
	}
	if mode == AuthModeOIDC {
		if cfg.Auth.OIDC.Issuer == "" {
			return fmt.Errorf("auth mode oidc requires --oidc-issuer")
		}
		if cfg.Auth.OIDC.JWKSURL == "" {
			return fmt.Errorf("auth mode oidc requires --oidc-jwks-url")
		}
	}
	if mode == AuthModeMTLS {
		if cfg.Auth.MTLS.CertFile == "" || cfg.Auth.MTLS.KeyFile == "" || cfg.Auth.MTLS.ClientCAFile == "" {
			return fmt.Errorf("auth mode mtls requires --mtls-cert, --mtls-key, and --mtls-ca")
		}
	}

	if mode == AuthModeLocal && !isLoopbackHost(cfg.Host) {
		return fmt.Errorf("refusing to bind %s without auth; set --auth-mode and required credentials", cfg.Host)
	}
	return nil
}

// New creates a new HTTP server.
func New(cfg Config) *Server {
	applyDefaults(&cfg)
	return &Server{
		host:               cfg.Host,
		port:               cfg.Port,
		eventBus:           cfg.EventBus,
		stateStore:         cfg.StateStore,
		auth:               cfg.Auth,
		sseClients:         make(map[chan events.BusEvent]struct{}),
		corsAllowedOrigins: cfg.AllowedOrigins,
		jwksCache:          newJWKSCache(cfg.Auth.OIDC.CacheTTL),
	}
}

// Start starts the HTTP server and blocks until shutdown.
func (s *Server) Start(ctx context.Context) error {
	if err := s.validate(); err != nil {
		return err
	}

	mux := http.NewServeMux()

	// REST API endpoints
	mux.HandleFunc("/api/sessions", s.handleSessions)
	mux.HandleFunc("/api/sessions/", s.handleSession)
	mux.HandleFunc("/api/robot/status", s.handleRobotStatus)
	mux.HandleFunc("/api/robot/health", s.handleRobotHealth)

	// SSE event stream
	mux.HandleFunc("/events", s.handleEventStream)
	// WebSocket stub (auth enforced even when implementation lands later)
	mux.HandleFunc("/ws", s.handleWS)

	// Health check
	mux.HandleFunc("/health", s.handleHealth)

	// Subscribe to events for SSE broadcasting
	if s.eventBus != nil {
		unsubscribe := s.eventBus.SubscribeAll(func(e events.BusEvent) {
			s.broadcastEvent(e)
		})
		defer unsubscribe()
	}

	s.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.host, s.port),
		Handler:      requestIDMiddleware(loggingMiddleware(s.corsMiddleware(s.authMiddleware(mux)))),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0, // Disabled to support long-lived SSE streams at /events
		IdleTimeout:  60 * time.Second,
	}

	scheme := "http"
	if s.auth.Mode == AuthModeMTLS {
		scheme = "https"
	}
	log.Printf("Starting NTM server on %s://%s:%d (auth=%s)", scheme, s.host, s.port, s.auth.Mode)

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		var err error
		if s.auth.Mode == AuthModeMTLS {
			tlsConfig, tlsErr := s.buildMTLSConfig()
			if tlsErr != nil {
				errCh <- tlsErr
				return
			}
			s.server.TLSConfig = tlsConfig
			err = s.server.ListenAndServeTLS(s.auth.MTLS.CertFile, s.auth.MTLS.KeyFile)
		} else {
			err = s.server.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		log.Println("Shutting down server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

// Port returns the configured port.
func (s *Server) Port() int {
	return s.port
}

func (s *Server) validate() error {
	cfg := Config{
		Host:           s.host,
		Port:           s.port,
		EventBus:       s.eventBus,
		StateStore:     s.stateStore,
		Auth:           s.auth,
		AllowedOrigins: s.corsAllowedOrigins,
	}
	applyDefaults(&cfg)
	mode, err := ParseAuthMode(string(cfg.Auth.Mode))
	if err != nil {
		return err
	}
	cfg.Auth.Mode = mode
	if err := ValidateConfig(cfg); err != nil {
		return err
	}
	s.host = cfg.Host
	s.port = cfg.Port
	s.auth = cfg.Auth
	s.corsAllowedOrigins = cfg.AllowedOrigins
	return nil
}

func (s *Server) buildMTLSConfig() (*tls.Config, error) {
	if s.auth.MTLS.CertFile == "" || s.auth.MTLS.KeyFile == "" || s.auth.MTLS.ClientCAFile == "" {
		return nil, fmt.Errorf("mtls requires cert, key, and client CA files")
	}
	caPEM, err := os.ReadFile(s.auth.MTLS.ClientCAFile)
	if err != nil {
		return nil, fmt.Errorf("read mtls CA: %w", err)
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("parse mtls CA: no certs found")
	}
	return &tls.Config{
		ClientCAs:  caPool,
		ClientAuth: tls.RequireAndVerifyClientCert,
		MinVersion: tls.VersionTLS12,
	}, nil
}

// requestIDMiddleware assigns a request ID and stores it in context and response headers.
func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get(requestIDHeader)
		if reqID == "" {
			reqID = generateRequestID()
		}
		w.Header().Set(requestIDHeader, reqID)
		ctx := context.WithValue(r.Context(), requestIDKey, reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// loggingMiddleware logs HTTP requests.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		reqID := requestIDFromContext(r.Context())
		if reqID != "" {
			log.Printf("%s %s %s request_id=%s", r.Method, r.URL.Path, time.Since(start), reqID)
			return
		}
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

// corsMiddleware adds CORS headers with an allowlist (default localhost).
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if !originAllowed(origin, s.corsAllowedOrigins) {
				writeError(w, http.StatusForbidden, "origin not allowed")
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, "+requestIDHeader)
		}

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// authMiddleware enforces configured authentication for all routes.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.auth.Mode == AuthModeLocal || s.auth.Mode == "" || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		if err := s.authenticateRequest(r); err != nil {
			reqID := requestIDFromContext(r.Context())
			log.Printf("auth failed mode=%s path=%s remote=%s request_id=%s err=%v", s.auth.Mode, r.URL.Path, r.RemoteAddr, reqID, err)
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) authenticateRequest(r *http.Request) error {
	switch s.auth.Mode {
	case AuthModeAPIKey:
		return s.authenticateAPIKey(r)
	case AuthModeOIDC:
		return s.authenticateOIDC(r)
	case AuthModeMTLS:
		return s.authenticateMTLS(r)
	case AuthModeLocal, "":
		return nil
	default:
		return fmt.Errorf("unsupported auth mode %q", s.auth.Mode)
	}
}

func (s *Server) authenticateAPIKey(r *http.Request) error {
	if s.auth.APIKey == "" {
		return errors.New("api key not configured")
	}
	key := extractAPIKey(r)
	if key == "" {
		return errors.New("missing api key")
	}
	if subtle.ConstantTimeCompare([]byte(key), []byte(s.auth.APIKey)) != 1 {
		return errors.New("invalid api key")
	}
	return nil
}

func (s *Server) authenticateOIDC(r *http.Request) error {
	token := extractBearerToken(r)
	if token == "" {
		return errors.New("missing bearer token")
	}
	return s.validateOIDCToken(r.Context(), token)
}

func (s *Server) authenticateMTLS(r *http.Request) error {
	if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
		return errors.New("missing client certificate")
	}
	return nil
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON: %v", err)
	}
}

// writeError writes an error response.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

// handleHealth handles health check requests.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"status":  "healthy",
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

// handleSessions handles /api/sessions - list all sessions.
func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if s.stateStore == nil {
		writeError(w, http.StatusServiceUnavailable, "state store not available")
		return
	}

	sessions, err := s.stateStore.ListSessions("")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":  true,
		"sessions": sessions,
		"count":    len(sessions),
	})
}

// handleSession handles /api/sessions/{id} - get session details.
func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract session ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusBadRequest, "session ID required")
		return
	}
	sessionID := parts[0]

	if s.stateStore == nil {
		writeError(w, http.StatusServiceUnavailable, "state store not available")
		return
	}

	session, err := s.stateStore.GetSession(sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if session == nil {
		writeError(w, http.StatusNotFound, "session not found")
		return
	}

	// Check for sub-resources
	if len(parts) > 1 {
		switch parts[1] {
		case "agents":
			s.handleSessionAgents(w, r, sessionID)
			return
		case "events":
			s.handleSessionEvents(w, r, sessionID)
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"session": session,
	})
}

// handleSessionAgents handles /api/sessions/{id}/agents.
func (s *Server) handleSessionAgents(w http.ResponseWriter, r *http.Request, sessionID string) {
	if s.stateStore == nil {
		writeError(w, http.StatusServiceUnavailable, "state store not available")
		return
	}

	agents, err := s.stateStore.ListAgents(sessionID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"session_id": sessionID,
		"agents":     agents,
		"count":      len(agents),
	})
}

// handleSessionEvents handles /api/sessions/{id}/events.
func (s *Server) handleSessionEvents(w http.ResponseWriter, r *http.Request, sessionID string) {
	if s.eventBus == nil {
		writeError(w, http.StatusServiceUnavailable, "event bus not available")
		return
	}

	// Get recent events from event bus history
	eventsData := s.eventBus.History(100)

	// Filter to session if specified
	var filtered []events.BusEvent
	for _, e := range eventsData {
		if sessionID == "" || e.EventSession() == sessionID {
			filtered = append(filtered, e)
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":    true,
		"session_id": sessionID,
		"events":     filtered,
		"count":      len(filtered),
	})
}

// handleRobotStatus handles /api/robot/status - proxies to robot status.
func (s *Server) handleRobotStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Return basic status - in a full implementation, this would call robot.Status()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"note":      "full robot status requires robot package integration",
	})
}

// handleRobotHealth handles /api/robot/health.
func (s *Server) handleRobotHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"note":      "full robot health requires robot package integration",
	})
}

// handleWS handles the WebSocket endpoint stub.
func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !isWebSocketUpgrade(r) {
		writeError(w, http.StatusBadRequest, "websocket upgrade required")
		return
	}
	writeError(w, http.StatusNotImplemented, "websocket hub not implemented")
}

// handleEventStream handles SSE event streaming at /events.
func (s *Server) handleEventStream(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Create client channel
	clientCh := make(chan events.BusEvent, 100)
	s.addSSEClient(clientCh)
	defer s.removeSSEClient(clientCh)

	// Get flusher for streaming
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	// Send initial connection event
	fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"connected\",\"time\":\"%s\"}\n\n",
		time.Now().UTC().Format(time.RFC3339))
	flusher.Flush()

	// Stream events
	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-clientCh:
			data, err := json.Marshal(map[string]interface{}{
				"type":      event.EventType(),
				"timestamp": event.EventTimestamp().Format(time.RFC3339),
				"session":   event.EventSession(),
			})
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.EventType(), data)
			flusher.Flush()
		}
	}
}

// addSSEClient adds a client to the SSE broadcast list.
func (s *Server) addSSEClient(ch chan events.BusEvent) {
	s.sseClientsMu.Lock()
	defer s.sseClientsMu.Unlock()
	s.sseClients[ch] = struct{}{}
}

// removeSSEClient removes a client from the SSE broadcast list.
func (s *Server) removeSSEClient(ch chan events.BusEvent) {
	s.sseClientsMu.Lock()
	defer s.sseClientsMu.Unlock()
	delete(s.sseClients, ch)
	close(ch)
}

// broadcastEvent sends an event to all SSE clients.
func (s *Server) broadcastEvent(event events.BusEvent) {
	s.sseClientsMu.RLock()
	defer s.sseClientsMu.RUnlock()

	for ch := range s.sseClients {
		select {
		case ch <- event:
		default:
			// Client buffer full, skip
		}
	}
}

func generateRequestID() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func requestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	val, ok := ctx.Value(requestIDKey).(string)
	if !ok {
		return ""
	}
	return val
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	parts := strings.Fields(auth)
	if len(parts) != 2 {
		return ""
	}
	if !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

func extractAPIKey(r *http.Request) string {
	if key := strings.TrimSpace(r.Header.Get("X-API-Key")); key != "" {
		return key
	}
	return extractBearerToken(r)
}

func isWebSocketUpgrade(r *http.Request) bool {
	upgrade := strings.ToLower(r.Header.Get("Upgrade"))
	if upgrade != "websocket" {
		return false
	}
	connection := strings.ToLower(r.Header.Get("Connection"))
	return strings.Contains(connection, "upgrade")
}

func originAllowed(origin string, allowlist []string) bool {
	if origin == "" {
		return true
	}
	if len(allowlist) == 0 {
		return false
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	for _, allowed := range allowlist {
		allowed = strings.TrimSpace(allowed)
		if allowed == "" {
			continue
		}
		if allowed == "*" {
			return true
		}
		if strings.Contains(allowed, "://") {
			allowedURL, err := url.Parse(allowed)
			if err != nil {
				continue
			}
			if strings.EqualFold(allowedURL.Scheme, u.Scheme) && strings.EqualFold(allowedURL.Hostname(), host) {
				if allowedURL.Port() == "" || allowedURL.Port() == u.Port() {
					return true
				}
			}
			continue
		}
		if strings.Contains(allowed, ":") {
			if strings.EqualFold(allowed, u.Host) {
				return true
			}
			continue
		}
		if strings.EqualFold(allowed, host) {
			return true
		}
	}
	return false
}

func isLoopbackHost(host string) bool {
	h := strings.TrimSpace(host)
	if h == "" {
		return true
	}
	if strings.EqualFold(h, "localhost") {
		return true
	}
	if strings.HasPrefix(h, "[") && strings.HasSuffix(h, "]") {
		h = strings.TrimPrefix(strings.TrimSuffix(h, "]"), "[")
	}
	if strings.Contains(h, ":") {
		if hostOnly, _, err := net.SplitHostPort(h); err == nil {
			h = hostOnly
		}
	}
	ip := net.ParseIP(h)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}

func (s *Server) validateOIDCToken(ctx context.Context, token string) error {
	if s.auth.OIDC.JWKSURL == "" || s.auth.OIDC.Issuer == "" {
		return errors.New("oidc config incomplete")
	}
	header, claims, signingInput, signature, err := parseJWT(token)
	if err != nil {
		return err
	}
	if header.Alg != "RS256" {
		return fmt.Errorf("unsupported jwt alg %q", header.Alg)
	}
	if iss, ok := claimString(claims, "iss"); !ok || iss != s.auth.OIDC.Issuer {
		return fmt.Errorf("invalid issuer")
	}
	if s.auth.OIDC.Audience != "" && !claimAudienceContains(claims, s.auth.OIDC.Audience) {
		return fmt.Errorf("invalid audience")
	}
	if exp, ok := claimInt64(claims, "exp"); ok {
		if time.Now().After(time.Unix(exp, 0).Add(30 * time.Second)) {
			return fmt.Errorf("token expired")
		}
	}
	if nbf, ok := claimInt64(claims, "nbf"); ok {
		if time.Now().Before(time.Unix(nbf, 0).Add(-30 * time.Second)) {
			return fmt.Errorf("token not yet valid")
		}
	}
	key, err := s.jwksCache.getKey(ctx, s.auth.OIDC.JWKSURL, header.Kid)
	if err != nil {
		return err
	}
	hash := sha256.Sum256([]byte(signingInput))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, hash[:], signature); err != nil {
		return fmt.Errorf("invalid token signature")
	}
	return nil
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
}

func parseJWT(token string) (jwtHeader, map[string]interface{}, string, []byte, error) {
	var header jwtHeader
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return header, nil, "", nil, fmt.Errorf("invalid jwt format")
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return header, nil, "", nil, fmt.Errorf("decode jwt header: %w", err)
	}
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return header, nil, "", nil, fmt.Errorf("decode jwt payload: %w", err)
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return header, nil, "", nil, fmt.Errorf("decode jwt signature: %w", err)
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return header, nil, "", nil, fmt.Errorf("parse jwt header: %w", err)
	}
	var claims map[string]interface{}
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return header, nil, "", nil, fmt.Errorf("parse jwt payload: %w", err)
	}
	return header, claims, parts[0] + "." + parts[1], signature, nil
}

func claimString(claims map[string]interface{}, key string) (string, bool) {
	raw, ok := claims[key]
	if !ok {
		return "", false
	}
	str, ok := raw.(string)
	return str, ok
}

func claimInt64(claims map[string]interface{}, key string) (int64, bool) {
	raw, ok := claims[key]
	if !ok {
		return 0, false
	}
	switch v := raw.(type) {
	case float64:
		return int64(v), true
	case json.Number:
		val, err := v.Int64()
		if err != nil {
			return 0, false
		}
		return val, true
	default:
		return 0, false
	}
}

func claimAudienceContains(claims map[string]interface{}, expected string) bool {
	raw, ok := claims["aud"]
	if !ok {
		return false
	}
	switch v := raw.(type) {
	case string:
		return v == expected
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok && s == expected {
				return true
			}
		}
	}
	return false
}

type jwksCache struct {
	mu        sync.Mutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
	ttl       time.Duration
}

func newJWKSCache(ttl time.Duration) *jwksCache {
	if ttl <= 0 {
		ttl = defaultJWKSCacheTTL
	}
	return &jwksCache{
		keys: make(map[string]*rsa.PublicKey),
		ttl:  ttl,
	}
}

func (c *jwksCache) getKey(ctx context.Context, jwksURL, kid string) (*rsa.PublicKey, error) {
	c.mu.Lock()
	if time.Since(c.fetchedAt) < c.ttl && len(c.keys) > 0 {
		if kid == "" && len(c.keys) == 1 {
			for _, key := range c.keys {
				c.mu.Unlock()
				return key, nil
			}
		}
		if key, ok := c.keys[kid]; ok {
			c.mu.Unlock()
			return key, nil
		}
	}
	c.mu.Unlock()

	keys, err := fetchJWKSKeys(ctx, jwksURL)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.keys = keys
	c.fetchedAt = time.Now()
	c.mu.Unlock()

	if kid == "" && len(keys) == 1 {
		for _, key := range keys {
			return key, nil
		}
	}
	key, ok := keys[kid]
	if !ok {
		return nil, fmt.Errorf("jwt kid not found in jwks")
	}
	return key, nil
}

type jwksPayload struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func fetchJWKSKeys(ctx context.Context, jwksURL string) (map[string]*rsa.PublicKey, error) {
	if jwksURL == "" {
		return nil, fmt.Errorf("jwks url missing")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build jwks request: %w", err)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch jwks: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fetch jwks: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read jwks: %w", err)
	}
	var payload jwksPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse jwks: %w", err)
	}
	keys := make(map[string]*rsa.PublicKey)
	for _, key := range payload.Keys {
		if key.Kty != "RSA" || key.N == "" || key.E == "" {
			continue
		}
		pub, err := parseRSAPublicKey(key.N, key.E)
		if err != nil {
			continue
		}
		kid := key.Kid
		if kid == "" {
			kid = "default"
		}
		keys[kid] = pub
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("no valid RSA keys in jwks")
	}
	return keys, nil
}

func parseRSAPublicKey(nStr, eStr string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, fmt.Errorf("decode jwk n: %w", err)
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, fmt.Errorf("decode jwk e: %w", err)
	}
	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)
	if e.Sign() <= 0 {
		return nil, fmt.Errorf("invalid jwk exponent")
	}
	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}, nil
}
