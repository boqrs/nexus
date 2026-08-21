package controller

import (
	"net/http"

	"github.com/boqrs/nexus"
	"github.com/gin-gonic/gin"
)

// Controller example
type Controller struct {
}

// NewController example
func NewController() *Controller {
	return &Controller{}
}

// Ping
// @Summary      Health check
// @Router /ping [get]
func (c *Controller) Ping(ginCtx *gin.Context) {
	traceID, _ := ginCtx.Get(comm.RequestTraceIDKey)
	spanID, _ := ginCtx.Get(comm.RequestSpanIDKey)

	ginCtx.JSON(http.StatusOK, gin.H{
		"message":  "pong",
		"trace_id": traceID,
		"span_id":  spanID,
	})
}
