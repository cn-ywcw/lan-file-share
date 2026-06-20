package server

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"
)

// CORS wraps a handler with CORS headers for LAN access.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// BasicAuth returns a middleware that enforces HTTP Basic Auth.
// Returns the original handler if authUser is empty.
func BasicAuth(authUser, authPass string, next http.HandlerFunc) http.HandlerFunc {
	if authUser == "" {
		return next
	}

	userHash := sha256.Sum256([]byte(authUser))
	passHash := sha256.Sum256([]byte(authPass))

	return func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok {
			AuthChallenge(w)
			return
		}
		uHash := sha256.Sum256([]byte(user))
		pHash := sha256.Sum256([]byte(pass))
		if subtle.ConstantTimeCompare(userHash[:], uHash[:]) != 1 ||
			subtle.ConstantTimeCompare(passHash[:], pHash[:]) != 1 {
			AuthChallenge(w)
			return
		}
		next(w, r)
	}
}

// AuthChallenge writes a 401 with WWW-Authenticate header to prompt browser login.
func AuthChallenge(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="LAN File Share"`)
	http.Error(w, "Authorization required", http.StatusUnauthorized)
}

// Logging wraps a handler with request logging.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only log API calls and non-static requests
		if strings.HasPrefix(r.URL.Path, "/api/") {
			lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(lrw, r)
			// Could log here if desired
		} else {
			next.ServeHTTP(w, r)
		}
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}
