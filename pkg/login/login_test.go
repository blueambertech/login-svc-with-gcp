package login

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/fatih/structs"
)

func TestMain(m *testing.M) {
	fc := FakeNoSQLClient{}
	fc.data = make(map[string]interface{})
	SetNoSQLClient(&fc)
	m.Run()
}

func TestAddLogin(t *testing.T) {
	ctx, canc := context.WithTimeout(context.Background(), time.Second*10)
	defer canc()
	err := AddLogin(ctx, "hello@test.com", "password")
	if err != nil {
		t.Error(err)
	}
}

func TestValidate(t *testing.T) {
	ctx, canc := context.WithTimeout(context.Background(), time.Second*10)
	defer canc()
	err := AddLogin(ctx, "hello@test.com", "password")
	if err != nil {
		t.Error(err)
	}

	result, err := Validate(ctx, "hello@test.com", "password")
	if err != nil {
		t.Error(err)
		return
	}
	if !result {
		t.Errorf("Result was false")
	}
}

func TestValidateWrongPass(t *testing.T) {
	ctx, canc := context.WithTimeout(context.Background(), time.Second*10)
	defer canc()
	err := AddLogin(ctx, "hello@test.com", "password")
	if err != nil {
		t.Error(err)
	}

	result, err := Validate(ctx, "hello@test.com", "passwfgdford")
	if err != nil {
		t.Error(err)
		return
	}
	if result {
		t.Errorf("Result should be false")
	}
}

type FakeNoSQLClient struct {
	data map[string]interface{}
}

func (f *FakeNoSQLClient) Read(_ context.Context, _, id string) (map[string]interface{}, error) {
	d := f.data[id].(*Details)
	return structs.Map(d), nil
}

func (f *FakeNoSQLClient) Insert(_ context.Context, _ string, data interface{}) (string, error) {
	id := fmt.Sprintf("%d", rand.New(rand.NewSource(535345)).Int())
	f.data[id] = data
	return id, nil
}

func (f *FakeNoSQLClient) InsertWithID(_ context.Context, _, id string, data interface{}) error {
	f.data[id] = data
	return nil
}

func (f *FakeNoSQLClient) Where(_ context.Context, _, _, _ string) ([]map[string]interface{}, error) {
	return nil, nil
}
