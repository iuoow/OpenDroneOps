package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iuoow/OpenDroneOps/internal/observability"
	"log/slog"
)

const maxAPIRequestBody = 1 << 20

type Server struct {
	httpServer  *http.Server
	adminServer *http.Server
	logger      *slog.Logger
	metrics     *observability.Registry
	handler     http.Handler
}

type RouteRegistrar interface {
	Register(*gin.RouterGroup)
}

type readinessProbe struct {
	name  string
	check func(context.Context) error
}

type options struct {
	adminAddr string
	metrics   *observability.Registry
	probes    []readinessProbe
}

type Option func(*options)

func WithAdminAddress(addr string) Option {
	return func(options *options) { options.adminAddr = addr }
}

func WithMetrics(registry *observability.Registry) Option {
	return func(options *options) { options.metrics = registry }
}

func WithReadinessProbe(name string, check func(context.Context) error) Option {
	return func(options *options) {
		if name != "" && check != nil {
			options.probes = append(options.probes, readinessProbe{name: name, check: check})
		}
	}
}

func New(addr string, logger *slog.Logger, registrars ...RouteRegistrar) *Server {
	return NewWithOptions(addr, logger, nil, registrars...)
}

func NewWithOptions(addr string, logger *slog.Logger, optionList []Option, registrars ...RouteRegistrar) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	settings := options{}
	for _, option := range optionList {
		option(&settings)
	}
	if settings.metrics == nil {
		settings.metrics = observability.NewRegistry(time.Now().UTC())
	}
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(securityHeaders(), requestID(), requestMetrics(settings.metrics))
	api := engine.Group("/api/v1")
	api.GET("/health/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "alive"})
	})
	api.GET("/health/ready", func(c *gin.Context) {
		for _, probe := range settings.probes {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
			err := probe.check(ctx)
			cancel()
			if err != nil {
				c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "dependency": probe.name})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
	for _, registrar := range registrars {
		if registrar != nil {
			registrar.Register(api)
		}
	}

	server := &Server{
		httpServer: &http.Server{
			Addr:              addr,
			Handler:           engine,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      15 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
		logger:  logger,
		metrics: settings.metrics,
		handler: engine,
	}
	if settings.adminAddr != "" {
		admin := http.NewServeMux()
		admin.Handle("/metrics", settings.metrics.Handler())
		server.adminServer = &http.Server{
			Addr:              settings.adminAddr,
			Handler:           admin,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      15 * time.Second,
			IdleTimeout:       60 * time.Second,
		}
	}
	return server
}

func (s *Server) Run() error {
	s.logger.Info("http server listening", "addr", s.httpServer.Addr)
	if s.adminServer != nil {
		s.logger.Info("admin server listening", "addr", s.adminServer.Addr)
		listener, err := net.Listen("tcp", s.adminServer.Addr)
		if err != nil {
			return fmt.Errorf("listen on admin address: %w", err)
		}
		go func() {
			if err := s.adminServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
				s.logger.Error("admin server stopped", "error", err)
			}
		}()
	}
	err := s.httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return err
	}
	if s.adminServer != nil {
		return s.adminServer.Shutdown(ctx)
	}
	return nil
}

func (s *Server) Metrics() *observability.Registry { return s.metrics }

func (s *Server) Handler() http.Handler { return s.handler }

func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Cache-Control", "no-store")
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAPIRequestBody)
		c.Next()
	}
}

func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if !validRequestID(requestID) {
			requestID = newRequestID()
		}
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Next()
	}
}

func requestMetrics(registry *observability.Registry) gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		registry.RecordHTTP(c.Request.Method, route, c.Writer.Status(), time.Since(started))
	}
}

func validRequestID(value string) bool {
	if len(value) == 0 || len(value) > 128 {
		return false
	}
	for _, character := range value {
		if character < 0x21 || character > 0x7e {
			return false
		}
	}
	return true
}

func newRequestID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err == nil {
		return hex.EncodeToString(bytes)
	}
	return fmt.Sprintf("generated-%d", time.Now().UnixNano())
}
