// Package contextx provides typed context accessors for {{.AppName | Title}}.
package contextx

import "context"

type (
	usernameKey   struct{}
	userIDKey     struct{}
	requestIDKey  struct{}
	accessTokenKey struct{}
	traceIDKey    struct{}
)

// WithUserID stores the user ID in the context.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey{}, userID)
}

// UserID retrieves the user ID from the context.
func UserID(ctx context.Context) string {
	v, _ := ctx.Value(userIDKey{}).(string)
	return v
}

// WithUsername stores the username in the context.
func WithUsername(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, usernameKey{}, username)
}

// Username retrieves the username from the context.
func Username(ctx context.Context) string {
	v, _ := ctx.Value(usernameKey{}).(string)
	return v
}

// WithRequestID stores the request ID in the context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

// RequestID retrieves the request ID from the context.
func RequestID(ctx context.Context) string {
	v, _ := ctx.Value(requestIDKey{}).(string)
	return v
}

// WithAccessToken stores the access token in the context.
func WithAccessToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, accessTokenKey{}, token)
}

// AccessToken retrieves the access token from the context.
func AccessToken(ctx context.Context) string {
	v, _ := ctx.Value(accessTokenKey{}).(string)
	return v
}

// WithTraceID stores the trace ID in the context.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey{}, traceID)
}

// TraceID retrieves the trace ID from the context.
func TraceID(ctx context.Context) string {
	v, _ := ctx.Value(traceIDKey{}).(string)
	return v
}
