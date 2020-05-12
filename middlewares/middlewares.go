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
	"wallet-adapter/utility"
)

var response = utility.NewResponse()

// Middleware ... Middleware struct
type Middleware struct {
	logger *utility.Logger
	config Config.Data
	next   http.HandlerFunc
}

// NewMiddleware ... Creates a middleware instance
func NewMiddleware(logger *utility.Logger, config Config.Data, handler http.HandlerFunc) *Middleware {
	return &Middleware{logger: logger, config: config, next: handler}
}

// Build ... Build midlleware functions
func (m *Middleware) Build() http.HandlerFunc {
	return m.next
}

// LogAPIRequests ... Logs every incoming request
func (m *Middleware) LogAPIRequests() *Middleware {
	nextHandler := http.HandlerFunc(func(responseWriter http.ResponseWriter, requestReader *http.Request) {
		m.logger.Info(fmt.Sprintf("Incoming request from : %s with IP : %s to : %s", requestReader.UserAgent(), utility.GetIPAdress(requestReader), requestReader.URL.Path))
		m.next.ServeHTTP(responseWriter, requestReader)
	})

	return &Middleware{logger: m.logger, config: m.config, next: nextHandler}
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
			json.NewEncoder(responseWriter).Encode(response.PlainError("TIMEOUT_ERR", utility.TIMEOUT_ERR))
			m.logger.Warning("Request Timeout: [duration = %f seconds.]", duration.Seconds())
			return
		}
	})

	// m.logger.Info("Timeout middleware registered successfully.")

	return &Middleware{logger: m.logger, config: m.config, next: nextHandler}
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
			m.logger.Error(fmt.Sprintf("Authentication token validation error %s", utility.EMPTY_AUTH_KEY))
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(responseWriter).Encode(response.PlainError("EMPTY_AUTH_KEY", utility.EMPTY_AUTH_KEY))
			return
		}

		if err := utility.VerifyJWT(authToken, m.config, &tokenClaims); err != nil {
			m.logger.Error(fmt.Sprintf("Authentication token validation error : %s", err))
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusForbidden)
			json.NewEncoder(responseWriter).Encode(response.PlainError("INVALID_AUTH_TOKEN", utility.INVALID_AUTH_TOKEN))
			return
		}

		if tokenClaims.TokenType != dto.JWT_TOKEN_TYPE.SERVICE {
			m.logger.Error(fmt.Sprintf("Authentication token validation error : %s", "Resource not accessible by non-service token type"))
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusForbidden)
			json.NewEncoder(responseWriter).Encode(response.PlainError("INVALID_AUTH_TOKEN", utility.INVALID_TOKENTYPE))
			return
		}

		if tokenClaims.ISS != dto.JWT_ISSUER {
			m.logger.Error(fmt.Sprintf("Authentication token validation error : %s", "Unknown Token Issuer"))
			responseWriter.Header().Set("Content-Type", "application/json")
			responseWriter.WriteHeader(http.StatusForbidden)
			json.NewEncoder(responseWriter).Encode(response.PlainError("INVALID_AUTH_TOKEN", utility.UNKNOWN_ISSUER))
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
		m.logger.Error(fmt.Sprintf("Authentication token validation error : %s", "Service does not have the required permission"))
		responseWriter.Header().Set("Content-Type", "application/json")
		responseWriter.WriteHeader(http.StatusForbidden)
		json.NewEncoder(responseWriter).Encode(response.Error("FORBIDDEN_ERR", utility.INVALID_PERMISSIONS, map[string]string{"permission": fmt.Sprintf("svcs.%s.%s", m.config.ServiceName, requiredPermission)}))
		return

	})

	return &Middleware{logger: m.logger, config: m.config, next: nextHandler}
}
