// Package known defines well-known constants for {{.AppName | Title}}.
//
// Prefer github.com/clin211/linhub/errx for standard HTTP header names used across linhub/core.
package known

import "github.com/clin211/linhub/errx"

// HTTP header constants (aligned with linhub errx / core).
const (
	// XRequestID is the header name for the unique request ID.
	XRequestID = errx.HeaderRequestID

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
