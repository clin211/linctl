package contextx

import (
	"context"
)

// 定义所有 context key 类型。
//
// 使用空结构体而不是字符串，可以避免 context key 命名冲突，并且零开销。
type (
	// usernameKey 是 username 在 context 中的 key。
	usernameKey struct{}
	// userIDKey 是 user ID 在 context 中的 key。
	userIDKey struct{}
	// accessTokenKey 是 access token 在 context 中的 key。
	accessTokenKey struct{}
	// requestIDKey 是 request ID 在 context 中的 key。
	requestIDKey struct{}
	// traceIDKey 是 trace ID 在 context 中的 key。
	traceIDKey struct{}
)

// WithUserID 把 user ID 存入 context 并返回新的 context。
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey{}, userID)
}

// UserID 从 context 中取出 user ID；不存在时返回空字符串。
func UserID(ctx context.Context) string {
	userID, _ := ctx.Value(userIDKey{}).(string)
	return userID
}

// WithUsername 把 username 存入 context 并返回新的 context。
func WithUsername(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, usernameKey{}, username)
}

// Username 从 context 中取出 username；不存在时返回空字符串。
func Username(ctx context.Context) string {
	username, _ := ctx.Value(usernameKey{}).(string)
	return username
}

// WithAccessToken 把 access token 存入 context 并返回新的 context。
func WithAccessToken(ctx context.Context, accessToken string) context.Context {
	return context.WithValue(ctx, accessTokenKey{}, accessToken)
}

// AccessToken 从 context 中取出 access token；不存在时返回空字符串。
func AccessToken(ctx context.Context) string {
	accessToken, _ := ctx.Value(accessTokenKey{}).(string)
	return accessToken
}

// WithRequestID 把 request ID 存入 context 并返回新的 context。
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

// RequestID 从 context 中取出 request ID；不存在时返回空字符串。
func RequestID(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDKey{}).(string)
	return requestID
}

// WithTraceID 把 trace ID 存入 context 并返回新的 context。
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey{}, traceID)
}

// TraceID 从 context 中取出 trace ID；不存在时返回空字符串。
func TraceID(ctx context.Context) string {
	traceID, _ := ctx.Value(traceIDKey{}).(string)
	return traceID
}
