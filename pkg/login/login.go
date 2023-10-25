package login

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/blueambertech/db"
	"github.com/blueambertech/pubsub"
	"github.com/mitchellh/mapstructure"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	hashIterations = 1000
	collectionName = "details"
	topicID        = "login-events"
)

type Details struct {
	UserName string
	PassHash string
	Salt     string
}

func (d *Details) String() string {
	if j, err := json.Marshal(d); err == nil {
		return string(j)
	}
	return "could not convert to string"
}

var dbClient db.NoSQLClient
var loginNotificationHandler pubsub.Handler

// SetNoSQLClient sets the NoSQL client to use to manage users
func SetNoSQLClient(client db.NoSQLClient) {
	dbClient = client
}

// SetNotificationQueue sets the queue handler to use for login notifications
func SetNotificationQueue(handler pubsub.Handler) {
	loginNotificationHandler = handler
}

// Validate takes a username and password and validates it against the details stored for this user in the login database
func Validate(ctx context.Context, userName, password string) (bool, error) {
	details, err := getDetails(ctx, userName)
	if status.Code(err) == codes.NotFound {
		return false, nil
	} else if err != nil {
		return false, err
	}
	hash := hashPassword(password + details.Salt)
	return hash == details.PassHash, nil
}

// AddLogin creates a new set of login details in the login database
func AddLogin(ctx context.Context, userName, password string) error {
	if dbClient == nil {
		return errors.New("database client not set")
	}
	salt, err := generateSalt()
	if err != nil {
		return err
	}

	d := Details{
		UserName: userName,
		PassHash: hashPassword(password + salt),
		Salt:     salt,
	}
	err = dbClient.InsertWithID(ctx, collectionName, d.UserName, &d)
	if err != nil {
		return err
	}
	// TODO: this isn't really a fail, in future add logging and return success
	return loginNotificationHandler.Push(ctx, topicID, "created login: "+userName)
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

func getDetails(ctx context.Context, userName string) (*Details, error) {
	if dbClient == nil {
		return nil, errors.New("database client not set")
	}
	record, err := dbClient.Read(ctx, collectionName, userName)
	if err != nil {
		return nil, err
	}

	var d Details
	err = mapstructure.Decode(record, &d)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func generateSalt() (string, error) {
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(salt), nil
}
