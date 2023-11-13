package login

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/fatih/structs"
	"github.com/mitchellh/mapstructure"
)

var fakeDbClient *FakeNoSQLClient
var fakeEventQueue *FakePubSubHandler

func TestMain(m *testing.M) {
	fakeDbClient = &FakeNoSQLClient{}
	fakeDbClient.data = make(map[string]map[string]interface{})
	fakeEventQueue = &FakePubSubHandler{}
	m.Run()
}

func TestAddLogin(t *testing.T) {
	defer clearData()
	ctx, canc := context.WithTimeout(context.Background(), time.Second*10)
	defer canc()
	err := AddLogin(ctx, fakeDbClient, fakeEventQueue, "hello@test.com", "password", nil)
	if err != nil {
		t.Error(err)
	}
}

func TestValidate(t *testing.T) {
	defer clearData()
	ctx, canc := context.WithTimeout(context.Background(), time.Second*10)
	defer canc()
	err := AddLogin(ctx, fakeDbClient, fakeEventQueue, "hello@test.com", "password", nil)
	if err != nil {
		t.Error(err)
	}

	result, _, err := Validate(ctx, fakeDbClient, "hello@test.com", "password")
	if err != nil {
		t.Error(err)
		return
	}
	if !result {
		t.Errorf("Result was false")
	}
}

func TestValidateWrongPass(t *testing.T) {
	defer clearData()
	ctx, canc := context.WithTimeout(context.Background(), time.Second*10)
	defer canc()
	err := AddLogin(ctx, fakeDbClient, fakeEventQueue, "hello@test.com", "password", nil)
	if err != nil {
		t.Error(err)
	}

	result, _, err := Validate(ctx, fakeDbClient, "hello@test.com", "passwfgdford")
	if err != nil {
		t.Error(err)
		return
	}
	if result {
		t.Errorf("Result should be false")
	}
}

func TestValidateAddLogin(t *testing.T) {
	defer clearData()
	result := ValidateAddLogin("valid@valid.com", "validpassword", nil)
	if !result {
		t.Error("incorrect validation of valid details")
		return
	}
	result = ValidateAddLogin("invaliduser", "validpassword", nil)
	if result {
		t.Error("incorrect validation of invalid username")
		return
	}
	result = ValidateAddLogin("valid@valid.com", "", nil)
	if result {
		t.Error("incorrect validation of invalid password")
		return
	}
}

func TestDetailsStringer(t *testing.T) {
	d := Details{
		UserName:    "Test",
		PassHash:    "hash",
		Salt:        "12345",
		DateCreated: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	expected := `{"UserName":"Test","PassHash":"hash","Salt":"12345","DateCreated":"2023-01-01T12:00:00Z"}`
	result := d.String()

	if result != expected {
		t.Errorf("stringer not produing correct result, expected %s got %s", expected, result)
	}
}

type FakeNoSQLClient struct {
	data map[string]map[string]interface{}
}

func (f *FakeNoSQLClient) Read(_ context.Context, _, id string) (map[string]interface{}, error) {
	d := f.data[id]
	return structs.Map(d), nil
}

func (f *FakeNoSQLClient) Insert(_ context.Context, _ string, data interface{}) (string, error) {
	id := fmt.Sprintf("%d", rand.New(rand.NewSource(535345)).Int())
	f.data[id] = structs.Map(data)
	return id, nil
}

func (f *FakeNoSQLClient) InsertWithID(_ context.Context, _, id string, data interface{}) error {
	f.data[id] = structs.Map(data)
	return nil
}

func (f *FakeNoSQLClient) Where(_ context.Context, _, _, _, val string) (map[string]map[string]interface{}, error) {
	var details = map[string]map[string]interface{}{}
	for i, v := range f.data {
		var d Details
		if err := mapstructure.Decode(v, &d); err != nil {
			return nil, err
		}
		// Assuming key is UserName and op is == for simplicity
		if d.UserName == val {
			details[i] = v
		}
	}
	return details, nil
}

func (f *FakeNoSQLClient) Exists(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

func clearData() {
	fakeDbClient.data = map[string]map[string]interface{}{}
}

type FakePubSubHandler struct {
}

func (pb *FakePubSubHandler) Subscribe(_ context.Context, _ string, _ time.Duration, _ func(c context.Context, msgData []byte)) error {
	return nil
}

func (pb *FakePubSubHandler) Push(_ context.Context, _, _ string) error {
	return nil
}
