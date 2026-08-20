package tracing

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// GinMiddleware 创建适用于 Gin 的链路追踪中间件
// serviceName: 用于标识 Tracer 的名称，通常与服务名一致
func GinMiddleware(serviceName string) gin.HandlerFunc {
	tracer := otel.Tracer(serviceName)
	propagator := otel.GetTextMapPropagator()

	return func(c *gin.Context) {
		ctx := c.Request.Context()

		// 1. 尝试使用标准 Propagator 提取 Context (支持 W3C, B3 等)
		ctx = propagator.Extract(ctx, propagation.HeaderCarrier(c.Request.Header))

		// 2. 【关键修复】修复短ID导致的链路断裂
		currentSpanCtx := trace.SpanContextFromContext(ctx)
		if currentSpanCtx.IsValid() {
			traceID := currentSpanCtx.TraceID().String()

			// 如果是 16 位，需要手动构建一个新的 32 位 SpanContext
			if len(traceID) == 16 {
				// 补零到 32 位
				normalizedTraceIDStr := fmt.Sprintf("%016s%s", "", traceID)
				normalizedTraceID, _ := trace.TraceIDFromHex(normalizedTraceIDStr)

				// 重新构建 SpanContext，使用标准化的 Trace ID
				newSpanCtx := trace.NewSpanContext(trace.SpanContextConfig{
					TraceID:    normalizedTraceID,
					SpanID:     currentSpanCtx.SpanID(),
					TraceFlags: currentSpanCtx.TraceFlags(),
					TraceState: currentSpanCtx.TraceState(),
					Remote:     currentSpanCtx.IsRemote(),
				})

				// 将新的 SpanContext 注入到 Context 中
				ctx = trace.ContextWithSpanContext(ctx, newSpanCtx)
				log.Printf("Normalized TraceID: %s -> %s", traceID, normalizedTraceIDStr)
			}
		}
		// 3. 创建 Span
		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}
		spanName := fmt.Sprintf("%s %s", c.Request.Method, route)

		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("http.method", c.Request.Method),
				attribute.String("http.url", c.Request.URL.String()),
				attribute.String("http.route", route),
				attribute.String("http.host", c.Request.Host),
				attribute.String("http.user_agent", c.Request.UserAgent()),
				attribute.String("http.client_ip", c.ClientIP()),
			),
		)
		defer span.End()

		// 4. 将包含 Span 的 Context 写回 Request
		c.Request = c.Request.WithContext(ctx)

		// 5. 记录开始时间
		startTime := time.Now()

		// 6. 处理请求
		c.Next()

		// 7. 记录响应信息
		latency := time.Since(startTime)
		statusCode := c.Writer.Status()

		span.SetAttributes(
			attribute.Int("http.status_code", statusCode),
			attribute.Int64("http.latency_ms", latency.Milliseconds()),
		)

		// 8. 设置 Span 状态
		if statusCode >= http.StatusInternalServerError {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d error", statusCode))
			for _, err := range c.Errors {
				span.RecordError(err.Err)
			}
		} else {
			span.SetStatus(codes.Ok, "success")
		}
	}
}
