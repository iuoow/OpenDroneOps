package trajectory

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type RouteRegistrar interface {
	Register(*gin.RouterGroup)
}

type Handler struct {
	store Store
	now   func() time.Time
}

func NewHandler(store Store) (*Handler, error) {
	if store == nil {
		return nil, errors.New("trajectory store is required")
	}
	return &Handler{store: store, now: func() time.Time { return time.Now().UTC() }}, nil
}

func (h *Handler) Register(api *gin.RouterGroup) {
	api.GET("/devices/:deviceID/trajectory", h.Get)
}

func (h *Handler) Get(c *gin.Context) {
	workspaceID := c.GetHeader("X-Workspace-ID")
	query := Query{
		WorkspaceID: workspaceID,
		DeviceID:    c.Param("deviceID"),
		Cursor:      c.Query("cursor"),
	}
	if value := c.Query("from"); value != "" {
		parsed, err := time.Parse(time.RFC3339Nano, value)
		if err != nil {
			writeQueryError(c, "from must be RFC3339")
			return
		}
		query.From = parsed
	}
	if value := c.Query("to"); value != "" {
		parsed, err := time.Parse(time.RFC3339Nano, value)
		if err != nil {
			writeQueryError(c, "to must be RFC3339")
			return
		}
		query.To = parsed
	}
	if value := c.Query("limit"); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil {
			writeQueryError(c, "limit must be an integer")
			return
		}
		query.Limit = limit
	}
	normalized, err := NormalizeQuery(query, h.now())
	if err != nil {
		writeQueryError(c, err.Error())
		return
	}
	page, err := h.store.Query(c.Request.Context(), normalized)
	if err != nil {
		if errors.Is(err, ErrInvalidQuery) || errors.Is(err, ErrInvalidCursor) {
			writeQueryError(c, err.Error())
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "trajectory query failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"items":       page.Items,
		"next_cursor": page.NextCursor,
		"truncated":   page.Truncated,
		"from":        normalized.From,
		"to":          normalized.To,
		"limit":       normalized.Limit,
	})
}

func writeQueryError(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, gin.H{
		"message": message,
		"code":    "INVALID_TRAJECTORY_QUERY",
	})
}
