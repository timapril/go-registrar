package lib

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
)

var bogusState = "bogus"

func Test_Core_GetRemoteUser(t *testing.T) {
	t.Parallel()

	request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	request.Header.Add("REMOTE_USER", TestUser1Username)

	username, err := GetRemoteUser(request)
	if username != TestUser1Username {
		t.Errorf("Expected username to be %s, got %s", TestUser1Username, username)
	}

	if err != nil {
		t.Errorf("Unexpected error when calling GetRemoteUser with valid username set: %s", err.Error())
	}

	request, _ = http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)

	_, err = GetRemoteUser(request)
	if err == nil {
		t.Error("Unexpected error when calling GetRemoteUser with no username set")
	}
}

func Test_Core_GetRemoteUserEmail(t *testing.T) {
	t.Parallel()

	// Create empty config with only the default user domain set
	conf := Config{}
	conf.Server.DefaultUserDomain = "example1.com"

	request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	request.Header.Add("REMOTE_USER", TestUser1Username)

	email, err := GetRemoteUserEmail(request, conf)
	if email != fmt.Sprintf("%s%s", TestUser1Username, "@example1.com") {
		t.Errorf("Expected email to be %s%s, got %s", TestUser1Username, "@example1.com", email)
	}

	if err != nil {
		t.Errorf("Unexpected error when calling GetRemoteUser with valid email set: %s", err.Error())
	}

	request, _ = http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	request.Header.Add("REMOTE_USER", TestUser1Username)
	request.Header.Add("REMOTE_USER_ORG", ExampleUserOrg)

	email, err = GetRemoteUserEmail(request, conf)
	if email != fmt.Sprintf("%s%s", TestUser1Username, ExampleUserOrg) {
		t.Errorf("Expected email to be %s%s, got %s", TestUser1Username, ExampleUserOrg, email)
	}

	if err != nil {
		t.Errorf("Unexpected error when calling GetRemoteUser with valid email set: %s", err.Error())
	}

	request, _ = http.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)

	_, err = GetRemoteUserEmail(request, conf)
	if err == nil {
		t.Error("Unexpected error when calling GetRemoteUser with no email set")
	}
}

func Test_NullInt64ConfirmEqual(t *testing.T) {
	t.Parallel()

	list := [][]sql.NullInt64{
		{
			{Int64: -1, Valid: false},
			{Int64: 0, Valid: false},
			{Int64: 1, Valid: false},
		},
		{{Int64: 0, Valid: true}},
		{{Int64: 1, Valid: true}},
		{{Int64: -1, Valid: true}},
	}

	for idx, aList := range list {
		for j, bList := range list {
			for _, a := range aList {
				for _, b := range bList {
					match, diff := NullInt64ConfirmEqual(a, b)

					if match != (idx == j) {
						t.Errorf("Error with NullInt64ConfirmEqual: %+v and %+v returned %t: %s", a, b, match, diff)
					}
				}
			}
		}
	}
}
