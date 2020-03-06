package util

import (
	"context"

	log "github.com/sirupsen/logrus"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func TraceLogger(ctx context.Context) *log.Entry {
	if ctx != nil {
		span, exists := tracer.SpanFromContext(ctx)
		if exists && span.Context().TraceID() != 0 {
			traceID := span.Context().TraceID()
			spanID := span.Context().SpanID()
			return log.WithFields(log.Fields{
				"dd.trace_id": traceID,
				"dd.span_id":  spanID,
			})
		}
	}
	// add a field to indicate that this logger doesn't have any context, and thus no traceID/spanID
	return log.WithField("dd.no_trace", "")
}
