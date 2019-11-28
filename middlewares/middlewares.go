package middlewares

import (
	"encoding/json"
	"fmt"
	"net/http"
	"wallet-adapter/utility"
)

var response = utility.NewResponse()

// Middleware ... Middleware struct
type Middleware struct {
	logger *utility.Logger
	next   http.Handler
}

// NewMiddleware ... Creates a middleware instance
func NewMiddleware(logger *utility.Logger, handler http.Handler) *Middleware {
	return &Middleware{logger, handler}
}

// Build ... Build midlleware functions
func (m *Middleware) Build() http.Handler {
	return m.next
}

var (
	serviceFunctions = []string{
		"create-assets",
		"get-assets",
	}
	X_API_KEY = "x-api-key"
)

// ValidateServiceAPIKey ... checks the header of incoming request for X_API_KEY and validate service permissions
func (m *Middleware) ValidateServiceAPIKey() *Middleware {
	nextHandler := http.HandlerFunc(func(responseWriter http.ResponseWriter, requestReader *http.Request) {
		apiKey := requestReader.Header.Get(X_API_KEY)
		if apiKey == "" {
			m.logger.Error(fmt.Sprintf("Outgoing response to %s : %s", requestReader.UserAgent(), "INVALID_API_KEY"))
			responseWriter.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(responseWriter).Encode(response.PlainError("INVALID_API_KEY", utility.INVALID_API_KEY))
			return
		}
		m.next.ServeHTTP(responseWriter, requestReader)
	})

	return &Middleware{m.logger, nextHandler}
}

// LogAPIRequests ... Logs every incoming request
func (m *Middleware) LogAPIRequests() *Middleware {
	nextHandler := http.HandlerFunc(func(responseWriter http.ResponseWriter, requestReader *http.Request) {
		m.logger.Info(fmt.Sprintf("Incoming request from : %s with IP : %s to : %s", requestReader.UserAgent(), utility.GetIPAdress(requestReader), requestReader.URL.Path))
		m.next.ServeHTTP(responseWriter, requestReader)
	})

	return &Middleware{m.logger, nextHandler}
}
