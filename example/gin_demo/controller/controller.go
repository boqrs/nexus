package controller

import (
	"net/http"

	"codeup.aliyun.com/65b21d33076e069afe3d3253/basice/comm"
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
