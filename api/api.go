package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"syscall"

	"github.com/blueambertech/logging"
	"github.com/blueambertech/login-svc-with-gcp/pkg/auth"
	"github.com/blueambertech/login-svc-with-gcp/pkg/login"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type LoginFormDetails struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

var ShutdownChannel chan os.Signal = make(chan os.Signal, 1)

// SetupHandlers sets up the http handlers for the required endpoints in this service using the default serve mux
func SetupHandlers() {
	http.HandleFunc("/shutdown", ShutdownHandler)
	http.HandleFunc("/health", HealthHandler)
	http.HandleFunc("/login/add", AddLoginHandler)
	http.HandleFunc("/login", LoginHandler)
	http.Handle("/testauth", auth.Authorize(http.HandlerFunc(TestAuthHandler))) // Use auth middleware here to authorize the user
}

// ShutdownHandler is a http handler that will gracefully shut the service down (in a real world environment this would require some authorisation)
func ShutdownHandler(w http.ResponseWriter, r *http.Request) {
	_, span := logging.Tracer.Start(r.Context(), "shutdown-span")
	defer span.End()
	log.Println("Service shutdown received")
	ShutdownChannel <- syscall.SIGTERM
}

// HealthHandler is a simple http handler that will return a 200 OK status, it can be used to check the service is up and running
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	_, span := logging.Tracer.Start(r.Context(), "health-span")
	defer span.End()
	w.WriteHeader(http.StatusOK)
}

// AddLoginHandler is a http handler that accepts a POST request to add a new login to the system
func AddLoginHandler(w http.ResponseWriter, r *http.Request) {
	_, span := logging.Tracer.Start(r.Context(), "add-login-request")
	defer span.End()

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	form, err := extractLoginFormDetails(r)
	if err != nil {
		httpError(w, "failed to extract form data", http.StatusBadRequest, span, err)
		return
	}

	err = login.AddLogin(r.Context(), form.Username, form.Password)
	if err != nil {
		httpError(w, "failed to add login", http.StatusBadRequest, span, err)
		return
	}
}

// LoginHandler is a http handler that accepts a POST request and verifies supplied credentials are valid, the response will contain a JWT which
// can be used to authenticate other requests
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	_, span := logging.Tracer.Start(r.Context(), "login-request")
	defer span.End()

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	form, err := extractLoginFormDetails(r)
	if err != nil {
		httpError(w, "failed to extract form data", http.StatusBadRequest, span, err)
		return
	}

	validCreds, err := login.Validate(r.Context(), form.Username, form.Password)
	if err != nil {
		httpError(w, "failed to validate", http.StatusInternalServerError, span, err)
		return
	}
	if !validCreds {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	token, err := auth.CreateJWT(r.Context(), form.Username)
	if err != nil {
		httpError(w, "failed to create token", http.StatusInternalServerError, span, err)
		return
	}
	_, _ = w.Write([]byte(token))
}

// TestAuthHandler is an example http handler that can be used to test requests are being authenticated correctly, it will be initialised using
// the auth middleware and should return the 200 OK status only if the JWT on the Authorization header is valid
func TestAuthHandler(w http.ResponseWriter, r *http.Request) {
	_, span := logging.Tracer.Start(r.Context(), "test-auth-span")
	defer span.End()
	w.WriteHeader(http.StatusOK)
}

func httpError(w http.ResponseWriter, msg string, httpStatus int, span trace.Span, err error) {
	http.Error(w, msg, httpStatus)
	span.RecordError(err)
	span.SetStatus(codes.Error, msg)
}

func extractLoginFormDetails(r *http.Request) (*LoginFormDetails, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	var form LoginFormDetails
	err = json.Unmarshal(body, &form)
	if err != nil {
		return nil, err
	}
	return &form, nil
}
