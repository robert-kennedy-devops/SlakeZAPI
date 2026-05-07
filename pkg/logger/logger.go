package logger

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"time"
)

type Level string

const (
	LevelDebug Level = "DEBUG"
	LevelInfo  Level = "INFO"
	LevelWarn  Level = "WARN"
	LevelError Level = "ERROR"
)

// contextKey is unexported to avoid collisions.
type contextKey string

const (
	keyRequestID contextKey = "request_id"
	keyTenantID  contextKey = "tenant_id"
	keyUserID    contextKey = "user_id"
)

// Logger is a minimal structured JSON logger.
type Logger struct {
	out    io.Writer
	fields map[string]interface{}
}

func New() *Logger {
	return &Logger{
		out:    os.Stdout,
		fields: make(map[string]interface{}),
	}
}

// WithField returns a new Logger with an extra field attached.
func (l *Logger) WithField(key string, value interface{}) *Logger {
	fields := make(map[string]interface{}, len(l.fields)+1)
	for k, v := range l.fields {
		fields[k] = v
	}
	fields[key] = value
	return &Logger{out: l.out, fields: fields}
}

// WithContext reads request_id and tenant_id from context.
func (l *Logger) WithContext(ctx context.Context) *Logger {
	lg := l
	if rid, ok := ctx.Value(keyRequestID).(string); ok && rid != "" {
		lg = lg.WithField("request_id", rid)
	}
	if tid, ok := ctx.Value(keyTenantID).(string); ok && tid != "" {
		lg = lg.WithField("tenant_id", tid)
	}
	if uid, ok := ctx.Value(keyUserID).(string); ok && uid != "" {
		lg = lg.WithField("user_id", uid)
	}
	return lg
}

func (l *Logger) log(level Level, msg string, extra map[string]interface{}) {
	entry := make(map[string]interface{}, len(l.fields)+len(extra)+3)
	entry["ts"] = time.Now().UTC().Format(time.RFC3339)
	entry["level"] = string(level)
	entry["msg"] = msg
	for k, v := range l.fields {
		entry[k] = v
	}
	for k, v := range extra {
		entry[k] = v
	}
	b, _ := json.Marshal(entry)
	l.out.Write(append(b, '\n'))
}

func (l *Logger) Info(msg string, fields ...map[string]interface{}) {
	l.log(LevelInfo, msg, merge(fields...))
}

func (l *Logger) Warn(msg string, fields ...map[string]interface{}) {
	l.log(LevelWarn, msg, merge(fields...))
}

func (l *Logger) Error(msg string, fields ...map[string]interface{}) {
	l.log(LevelError, msg, merge(fields...))
}

func (l *Logger) Debug(msg string, fields ...map[string]interface{}) {
	l.log(LevelDebug, msg, merge(fields...))
}

func (l *Logger) Audit(action string, fields ...map[string]interface{}) {
	extra := merge(fields...)
	extra["event_type"] = "audit"
	extra["action"] = action
	l.log(LevelInfo, "audit", extra)
}

func merge(maps ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

// ─── Context helpers ─────────────────────────────────────────────────────────

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, keyRequestID, id)
}

func WithTenantID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, keyTenantID, id)
}

func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, keyUserID, id)
}

func RequestIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(keyRequestID).(string)
	return v
}

func TenantIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(keyTenantID).(string)
	return v
}

func UserIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(keyUserID).(string)
	return v
}
