package middlewares

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	Config "wallet-adapter/config"
	"wallet-adapter/dto"
	"wallet-adapter/errorcode"
	"wallet-adapter/utility"
	"wallet-adapter/utility/logger"
)

var response = utility.NewResponse()

// Middleware ... Middleware struct
type Middleware struct {
	config Config.Data
	next   http.HandlerFunc
}

// NewMiddleware ... Creates a middleware instance
func NewMiddleware(config Config.Data, handler http.HandlerFunc) *Middleware {
	return &Middleware{config: config, next: handler}
}

// Build ... Build midlleware functions
func (m *Middleware) Build() http.HandlerFunc {
	return m.next
}

// LogAPIRequests ... Logs every incoming request
func (m *Middleware) LogAPIRequests() *Middleware {
	nextHandler := http.HandlerFunc(func(responseWriter http.ResponseWriter, requestReader *http.Request) {
		logger.Info(fmt.Sprintf("Incoming request from : %s with IP : %s to : %s", requestReader.UserAgent(), utility.GetIPAdress(requestReader), requestReader.URL.Path))
		m.next.ServeHTTP(responseWriter, requestReader)
	})

	return &Middleware{config: m.config, next: nextHandler}
}

// Timeout cancels a slow request after a given duration
func (m *Middleware) Timeout(duration time.Duration) *Middleware {
	AttemptNextRequest := func(responseWriter http.ResponseWriter, requestReader *http.Request) <-chan struct{} {
		completed := make(chan struct{})

		go func(responseWriter *http.ResponseWriter, requestReader *http.Request) {
			m.next.ServeHTTP(*responseWriter, requestReader)
			completed <- struct{}{}
		}(&responseWriter, requestReader)

		return completed
	}

	nextHandler := http.HandlerFunc(func(responseWriter http.ResponseWriter, requestReader *http.Request) {
		ctx, releaseContext := context.WithTimeout(requestReader.Context(), duration)
		defer releaseContext()

		nextRequestCompleted := AttemptNextRequest(responseWriter, requestReader.WithContext(ctx))

		select {
		case <-nextRequestCompleted:
			break
		case <-ctx.Done():
			json.NewEncoder(responseWriter).Encode(response.PlainError("TIMEOUT_ERR", errorcode.TIMEOUT_ERR))
			logger.Warning("Request Timeout: [duration = %f seconds.]", duration.Seconds())
			return
		}
	})

	// logger.Info("Timeout middleware registered successfully.")

	return &Middleware{config: m.config, next: nextHandler}
}

// ValidateAuthToken ... retrieves auth toke from header                                                                                                                 and Verifies token permissions
func (m *Middleware) ValidateAuthToken(requiredPermission string) *Middleware {

	nextHandler := http.HandlerFunc(func(responseWriter http.ResponseWriter, requestReader *http.Request) {
		if strings.Contains(requestReader.URL.Path, "swagger") {
			m.next.ServeHTTP(responseWriter, requestReader)
			return
		}

		authToken := requestReader.Header.Get(utility.X_AUTH_TOKEN)
		tokenClaims := dto.TokenClaims{}

		if authToken == "" {
			logger.Error(fmt.Sprintf("Authentication token validation error %s", errorcode.EMPTY_AUTH_KEY))
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(responseWriter).Encode(response.PlainError("EMPTY_AUTH_KEY", errorcode.EMPTY_AUTH_KEY))
			return
		}

		if err := utility.VerifyJWT(authToken, m.config, &tokenClaims); err != nil {
			logger.Error(fmt.Sprintf("Authentication token validation error : %s", err))
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusForbidden)
			json.NewEncoder(responseWriter).Encode(response.PlainError("INVALID_AUTH_TOKEN", errorcode.INVALID_AUTH_TOKEN))
			return
		}

		if tokenClaims.TokenType != dto.JWT_TOKEN_TYPE.SERVICE {
			logger.Error(fmt.Sprintf("Authentication token validation error : %s", "Resource not accessible by non-service token type"))
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusForbidden)
			json.NewEncoder(responseWriter).Encode(response.PlainError("INVALID_AUTH_TOKEN", errorcode.INVALID_TOKENTYPE))
			return
		}

		if tokenClaims.ISS != dto.JWT_ISSUER {
			logger.Error(fmt.Sprintf("Authentication token validation error : %s", "Unknown Token Issuer"))
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusForbidden)
			json.NewEncoder(responseWriter).Encode(response.PlainError("INVALID_AUTH_TOKEN", errorcode.UNKNOWN_ISSUER))
			return
		}

		for i := 0; i < len(tokenClaims.Permissions); i++ {
			permissionSlice := strings.Split(tokenClaims.Permissions[i], ".")
			serviceName := permissionSlice[len(permissionSlice)-2]
			permission := permissionSlice[len(permissionSlice)-1]
			if serviceName == m.config.ServiceName && permission == requiredPermission {
				m.next.ServeHTTP(responseWriter, requestReader)
				return
			}
		}
		logger.Error(fmt.Sprintf("Authentication token validation error : %s", "Service does not have the required permission"))
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusForbidden)
		json.NewEncoder(responseWriter).Encode(response.Error("FORBIDDEN_ERR", errorcode.INVALID_PERMISSIONS, map[string]string{"permission": fmt.Sprintf("svcs.%s.%s", m.config.ServiceName, requiredPermission)}))
		return

	})

	return &Middleware{config: m.config, next: nextHandler}
}
