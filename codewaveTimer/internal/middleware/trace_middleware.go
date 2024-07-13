package middleware

import (
	"codewave-timer/codewaveTimer/pkg/trace"
	"net/http"
)

func TraceMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 从请求头中获取 traceID
		traceID := r.Header.Get("X-Trace-ID")

		// 创建新的上下文和 Span
		ctx, span := tracing.NewSpan(tracing.WithTraceID(r.Context(), traceID), "http_request", r.URL.Path)
		defer span.End()

		// 将上下文传递给下一个 handler
		r = r.WithContext(ctx)

		// 使用自定义的响应写入器
		wrappedWriter := NewResponseWriter(w)
		next(wrappedWriter, r)

		// 在响应头中添加 traceID
		wrappedWriter.Header().Set("X-Trace-ID", span.SpanContext().TraceID().String())
	}
}

// ResponseWriter 是一个自定义的响应写入器，能够在响应头中添加额外的字段
type ResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

// NewResponseWriter 创建一个新的 ResponseWriter
func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{w, http.StatusOK}
}

// WriteHeader 覆盖默认的 WriteHeader 方法以捕获状态码
func (rw *ResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// StatusCode 获取捕获的状态码
func (rw *ResponseWriter) StatusCode() int {
	return rw.statusCode
}
