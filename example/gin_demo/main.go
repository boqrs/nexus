package main

import (
	"context"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"google.golang.org/grpc/credentials"
)

func main() {
	ctx := context.Background()

	// 1. 创建 Exporter
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint("tracing-analysis-dc-hz-internal.aliyuncs.com:8090"), // 替换为你的 Endpoint
		otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, "")),
		otlptracegrpc.WithHeaders(map[string]string{
			"x-arms-license-key": "ku8jh@f615468cee206ab_ku8jh@53df7ad2afe8301", // 替换为你的 Key
		}),
		otlptracegrpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		log.Fatalf("failed to create exporter: %v", err)
	}

	// 2. 创建 Provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceNameKey.String("test-service"))),
	)
	otel.SetTracerProvider(tp)

	// 3. 产生一个 Span
	tracer := otel.Tracer("test")
	_, span := tracer.Start(ctx, "test-span")
	span.End()

	// 4. 强制刷新并关闭
	if err := tp.Shutdown(ctx); err != nil {
		log.Fatalf("failed to shutdown: %v", err)
	}
	log.Println("Test finished. Check ARMS console.")
}
