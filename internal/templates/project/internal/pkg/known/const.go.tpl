// Package known defines well-known constants for {{.AppName | Title}}.
package known

// HTTP header constants (lowercase for HTTP/2 and gRPC compatibility).
const (
	// XRequestID is the header name for the unique request ID.
	XRequestID = "x-request-id"

	// XUserID is the header name for the authenticated user ID.
	XUserID = "x-user-id"

	// XUsername is the header name for the authenticated username.
	XUsername = "x-username"
)

// Application constants.
const (
	// AdminUsername is the default admin user.
	AdminUsername = "root"

	// MaxErrGroupConcurrency limits goroutine concurrency in errgroup.
	MaxErrGroupConcurrency = 1000
)
