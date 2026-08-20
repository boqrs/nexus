package comm

// Request trace/span keys for Gin context values.
//
// These keys are used by middleware to store trace information for downstream
// handlers (e.g. logging, tracing, metrics).
const (
	RequestTraceIDKey = "trace_id"
	RequestSpanIDKey  = "span_id"
)
