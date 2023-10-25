package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/blueambertech/firestoredb"
	"github.com/blueambertech/googlepubsub"
	"github.com/blueambertech/logging"
	"github.com/blueambertech/login-svc-with-gcp/api"
	"github.com/blueambertech/login-svc-with-gcp/data"
	"github.com/blueambertech/login-svc-with-gcp/pkg/login"
)

const dbName = "<GCP Firestore Database Name here>"

func main() {
	bgCtx := context.Background()
	logging.Setup(bgCtx, data.ServiceName)

	port := os.Getenv("PORT") // PORT set by GCP when running in their serverless environment
	if port == "" {
		port = "8081"
	}
	server := &http.Server{
		Addr: ":" + port,
	}

	db, err := firestoredb.NewFirestore(data.ProjectID, dbName)
	if err != nil {
		log.Fatal(err)
	}
	login.SetNoSQLClient(db)

	pubsub, err := googlepubsub.New(bgCtx, data.ProjectID)
	if err != nil {
		log.Fatal(err)
	}
	login.SetNotificationQueue(pubsub)

	api.SetupHandlers()

	go func() {
		log.Println("Service started")
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Println("HTTP server error:", err)
			api.ShutdownChannel <- syscall.SIGKILL
		}
		log.Println("Stopped serving new connections")
	}()

	waitForShutdown(server)
}

func waitForShutdown(server *http.Server) {
	signal.Notify(api.ShutdownChannel, syscall.SIGINT, syscall.SIGTERM)
	<-api.ShutdownChannel

	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Service shutdown error: %v", err)
	}
	log.Println("Service shutdown complete")
}
