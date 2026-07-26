package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"log/slog"
)

type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
}

type RouteRegistrar interface {
	Register(*gin.RouterGroup)
}

func New(addr string, logger *slog.Logger, registrars ...RouteRegistrar) *Server {
	engine := gin.New()
	engine.Use(gin.Recovery())
	api := engine.Group("/api/v1")
	api.GET("/health/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "alive"})
	})
	api.GET("/health/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
	for _, registrar := range registrars {
		if registrar != nil {
			registrar.Register(api)
		}
	}

	return &Server{
		httpServer: &http.Server{
			Addr:              addr,
			Handler:           engine,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      15 * time.Second,
			IdleTimeout:       60 * time.Second,
		},
		logger: logger,
	}
}

func (s *Server) Run() error {
	s.logger.Info("http server listening", "addr", s.httpServer.Addr)
	err := s.httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
