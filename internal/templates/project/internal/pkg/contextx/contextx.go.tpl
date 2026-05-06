package contextx

import "context"

type (
	usernameKey   struct{}
	userIDKey     struct{}
	requestIDKey  struct{}
	accessTokenKey struct{}
	traceIDKey    struct{}
)

// WithUserID 将用户 ID 存入 context。
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey{}, userID)
}

// UserID 从 context 中取出用户 ID。
func UserID(ctx context.Context) string {
	v, _ := ctx.Value(userIDKey{}).(string)
	return v
}

// WithUsername 将用户名存入 context。
func WithUsername(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, usernameKey{}, username)
}

// Username 从 context 中取出用户名。
func Username(ctx context.Context) string {
	v, _ := ctx.Value(usernameKey{}).(string)
	return v
}

// WithRequestID 将请求 ID 存入 context。
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

// RequestID 从 context 中取出请求 ID。
func RequestID(ctx context.Context) string {
	v, _ := ctx.Value(requestIDKey{}).(string)
	return v
}

// WithAccessToken 将访问令牌存入 context。
func WithAccessToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, accessTokenKey{}, token)
}

// AccessToken 从 context 中取出访问令牌。
func AccessToken(ctx context.Context) string {
	v, _ := ctx.Value(accessTokenKey{}).(string)
	return v
}

// WithTraceID 将 trace ID 存入 context。
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey{}, traceID)
}

// TraceID 从 context 中取出 trace ID。
func TraceID(ctx context.Context) string {
	v, _ := ctx.Value(traceIDKey{}).(string)
	return v
}
