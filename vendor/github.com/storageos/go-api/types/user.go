package types

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
)

// User represents a created user
type User struct {
	UUID string `json:"id"`

	UserCreateOptions // expand inner fields
}

// UserCreateOptions is used to provide information for user creation
type UserCreateOptions struct {
	Username string   `json:"username"`
	Groups   commaStr `json:"groups"`
	Password string   `json:"password"`
	Role     string   `json:"role"`

	// Context can be set with a timeout or can be used to cancel a request.
	Context context.Context `json:"-"`
}

// commaStr can unmarshal both JSON string array and a comma separated string
type commaStr []string

// UnmarshalJSON implements json.Unmarshaller
func (s *commaStr) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil
	}

	if data[0] == '"' {
		*s = strings.Split(strings.Trim(string(data), `"`), ",")
		return nil
	}
	ss := []string{}
	if err := json.Unmarshal(data, &ss); err != nil {
		return err
	}
	*s = ss
	return nil
}
