package auth

import (
	"context"
	"testing"
)

func TestCreateJWT(t *testing.T) {
	jwt, e := CreateJWT(context.Background(), "testuser")
	if e != nil {
		t.Error(e)
		return
	}
	if len(jwt) <= 0 {
		t.Error("Generated token is empty")
		return
	}
}
