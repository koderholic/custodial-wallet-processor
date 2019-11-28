package middlewares

import (
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

// LogAPIRequests ... Logs every incoming request
func (m *Middleware) LogAPIRequests() *Middleware {
	nextHandler := http.HandlerFunc(func(responseWriter http.ResponseWriter, requestReader *http.Request) {
		m.logger.Info(fmt.Sprintf("Incoming request from : %s with IP : %s to : %s", requestReader.UserAgent(), utility.GetIPAdress(requestReader), requestReader.URL.Path))
		m.next.ServeHTTP(responseWriter, requestReader)
	})

	return &Middleware{m.logger, nextHandler}
}
