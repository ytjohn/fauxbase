package middleware

// Add at package level
type contextKey string

const (
	UserContextKey    contextKey = "user"
	LoggerKey         contextKey = "logger"
	KeyManagerKey     contextKey = "keyManager"
	ResponseWriterKey contextKey = "response_writer"
	TokenServiceKey   contextKey = "tokenService"
	TokenContextKey   contextKey = "token"
)
