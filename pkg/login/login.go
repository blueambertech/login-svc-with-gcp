package login

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/blueambertech/db"
	"github.com/blueambertech/login-svc-with-gcp/pkg/verification"
	"github.com/blueambertech/pubsub"

	"github.com/mitchellh/mapstructure"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	hashIterations = 1000
	collectionName = "details"
	topicID        = "login-events"
)

type Details struct {
	UserName    string
	PassHash    string
	Salt        string
	DateCreated time.Time
}

func (d *Details) String() string {
	if j, err := json.Marshal(d); err == nil {
		return string(j)
	}
	return "could not convert to string"
}

// Validate takes a username and password and validates it against the details stored for this user in the login database,
// it also returns the user ID
func Validate(ctx context.Context, dbClient db.NoSQLClient, userName, password string) (bool, string, error) {
	details, id, err := getDetails(ctx, dbClient, userName)
	if status.Code(err) == codes.NotFound {
		return false, "", nil
	} else if err != nil {
		return false, "", err
	}
	hash := hashPassword(password + details.Salt)
	return hash == details.PassHash, id, nil
}

// ValidateAddLogin validates the provided login details are ok to insert
func ValidateAddLogin(userName, password string, traceSpan trace.Span) bool {
	if !verification.VerifyEmail(userName) {
		if traceSpan != nil {
			traceSpan.AddEvent("email invalid: " + userName)
		}
		return false
	}
	return len(password) > 0
}

// AddLogin creates a new set of login details in the login database
func AddLogin(ctx context.Context, dbClient db.NoSQLClient, eventQueue pubsub.Handler, userName, password string, traceSpan trace.Span) error {
	// Check doesn't exist (user name must be unique)
	docs, err := dbClient.Where(ctx, collectionName, "UserName", "==", userName)
	if err != nil {
		return err
	}
	if len(docs) > 0 {
		return errors.New("a user already exists with this username")
	}

	salt, err := generateSalt()
	if err != nil {
		return err
	}

	d := Details{
		UserName:    userName,
		PassHash:    hashPassword(password + salt),
		Salt:        salt,
		DateCreated: time.Now(),
	}

	id, err := dbClient.Insert(ctx, collectionName, &d)
	if err != nil {
		return err
	}
	if err = eventQueue.Push(ctx, topicID, "created: "+id); err != nil {
		if traceSpan != nil {
			traceSpan.AddEvent("failed to push login notification to queue")
		}
	}
	return nil
}

func hashPassword(pw string) string {
	hp := []byte(pw)
	for i := 0; i < hashIterations; i++ {
		h := sha256.New()
		h.Write(hp)
		hp = h.Sum(nil)
	}
	return fmt.Sprintf("%x", hp)
}

func getDetails(ctx context.Context, dbClient db.NoSQLClient, userName string) (*Details, string, error) {
	records, err := dbClient.Where(ctx, collectionName, "UserName", "==", userName)
	if err != nil {
		return nil, "", err
	}
	if len(records) == 0 {
		return nil, "", errors.New("no user found with this username")
	}
	if len(records) > 1 {
		// TODO: Raise a warning message
	}

	var d Details
	var id string
	for i, val := range records {
		err = mapstructure.Decode(val, &d)
		if err != nil {
			return nil, "", err
		}
		id = i
		break
	}
	return &d, id, nil
}

func generateSalt() (string, error) {
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(salt), nil
}
