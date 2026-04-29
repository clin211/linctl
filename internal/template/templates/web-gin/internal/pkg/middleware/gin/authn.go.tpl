package gin

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"

	"{{ .Project.Metadata.Module }}/internal/pkg/contextx"
	"{{ .Project.Metadata.Module }}/internal/pkg/errno"
	"{{ .Project.Metadata.Module }}/pkg/core"
	"{{ .Project.Metadata.Module }}/pkg/token"
)

// UserRetriever fetches user info by ID. Returning (any, nil) is enough to
// indicate the user exists; the middleware does not introspect the value.
//
// Implementations are encouraged to enrich the request context (e.g., via
// `contextx.WithUsername`) before returning, but it is not mandatory.
type UserRetriever interface {
	GetUser(ctx context.Context, userID string) (any, error)
}

// AuthnMiddleware extracts and validates the JWT from the request, then
// loads the user via the supplied UserRetriever.
//
// On success, contextx.UserID is populated for downstream handlers and the
// AuthzMiddleware. Failures abort the chain with `errno.ErrTokenInvalid` /
// `errno.ErrUnauthenticated`.
func AuthnMiddleware(retriever UserRetriever) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := token.ParseRequest(c)
		if err != nil {
			core.WriteResponse(c, nil, errno.ErrTokenInvalid.WithMessage(err.Error()))
			c.Abort()
			return
		}

		slog.Info("Token parsing successful", "userID", userID)

		if _, err = retriever.GetUser(c, userID); err != nil {
			core.WriteResponse(c, nil, errno.ErrUnauthenticated.WithMessage(err.Error()))
			c.Abort()
			return
		}

		ctx := contextx.WithUserID(c.Request.Context(), userID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
