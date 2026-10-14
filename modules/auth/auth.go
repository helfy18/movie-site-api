package auth

import (
	"context"
	"net/http"
	"strings"

	firebase "firebase.google.com/go/v4"
	fbauth "firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
)

// contextKey is where RequireAuth stores the verified token on the gin context.
const contextKey = "firebaseToken"

/*
NewVerifier returns a Firebase auth client for the given project. Verifying ID
tokens only needs the project ID and Google's published public keys (which the
SDK fetches and caches itself), so no service-account credentials are required.
*/
func NewVerifier(ctx context.Context, projectID string) (*fbauth.Client, error) {
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: projectID})
	if err != nil {
		return nil, err
	}
	return app.Auth(ctx)
}

/*
RequireAuth is gin middleware that verifies the Firebase ID token in the
Authorization header (as "Bearer <token>") and aborts with 401 if it is
missing or invalid. Handlers behind it read the result with User(c).
*/
func RequireAuth(client *fbauth.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		tokenString, found := strings.CutPrefix(header, "Bearer ")
		if !found || tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing bearer token"})
			return
		}

		token, err := client.VerifyIDToken(c.Request.Context(), tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		c.Set(contextKey, token)
		c.Next()
	}
}

// User returns the verified token set by RequireAuth, or nil when the route
// is not behind the middleware.
func User(c *gin.Context) *fbauth.Token {
	if token, ok := c.Get(contextKey); ok {
		return token.(*fbauth.Token)
	}
	return nil
}

/*
DisplayName extracts a human-readable name from the token's claims: the
provider profile name when present, otherwise the part of the email before
the @, otherwise "Anonymous".
*/
func DisplayName(token *fbauth.Token) string {
	if name, ok := token.Claims["name"].(string); ok && name != "" {
		return name
	}
	if email, ok := token.Claims["email"].(string); ok && email != "" {
		if at := strings.Index(email, "@"); at > 0 {
			return email[:at]
		}
	}
	return "Anonymous"
}
