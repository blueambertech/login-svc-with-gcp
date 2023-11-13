package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"syscall"

	"github.com/blueambertech/db"
	"github.com/blueambertech/httpauth"
	"github.com/blueambertech/logging"
	"github.com/blueambertech/login-svc-with-gcp/pkg/login"
	"github.com/blueambertech/pubsub"
	"github.com/blueambertech/secretmanager"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type LoginFormDetails struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

var (
	ShutdownChannel chan os.Signal = make(chan os.Signal, 1)
	secrets         secretmanager.SecretManager
	dbClient        db.NoSQLClient
	events          pubsub.Handler
)

// SetupHandlers sets up the http handlers for the required endpoints in this service using the default serve mux
func SetupHandlers(sm secretmanager.SecretManager, db db.NoSQLClient, eq pubsub.Handler) {
	secrets = sm
	dbClient = db
	events = eq
	http.HandleFunc("/health", HealthHandler)
	http.HandleFunc("/login/add", AddLoginHandler)
	http.HandleFunc("/login", LoginHandler)
	http.Handle("/shutdown", httpauth.Authorize(http.HandlerFunc(ShutdownHandler), secrets))
	http.Handle("/testauth", httpauth.Authorize(http.HandlerFunc(TestAuthHandler), secrets))
}

// ShutdownHandler is a http handler that will gracefully shut the service down
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

	err = login.AddLogin(r.Context(), dbClient, events, form.Username, form.Password, span)
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

	validCreds, _, err := login.Validate(r.Context(), dbClient, form.Username, form.Password)
	if err != nil {
		httpError(w, "failed to validate", http.StatusInternalServerError, span, err)
		return
	}
	if !validCreds {
		w.WriteHeader(http.StatusForbidden)
		return
	}
	claims := map[string]interface{}{
		"exp": httpauth.StandardTokenLife,
	}
	token, err := httpauth.CreateJWTWithClaims(r.Context(), secrets, claims)
	if err != nil {
		httpError(w, "failed to create token", http.StatusInternalServerError, span, err)
		return
	}
	_, _ = w.Write([]byte(token))
}

// TestAuthHandler is an example http handler that can be used to test requests are being authenticated correctly, it should be initialised using
// the auth middleware which will return the 200 OK status only if the JWT on the Authorization header is valid
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
