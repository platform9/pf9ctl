package pmk

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"time"

	"go.uber.org/zap"
)

// Context stores information to contact with the pf9 controller.
type Context struct {
	Fqdn          string        `json:"fqdn"`
	Username      string        `json:"os_username"`
	Password      string        `json:"os_password"`
	Tenant        string        `json:"os_tenant"`
	Region        string        `json:"os_region"`
	WaitPeriod    time.Duration `json:"wait_period"`
	AllowInsecure bool          `json:"allow_insecure"`
}

// StoreContext simply updates the in-memory object
func StoreContext(ctx Context, loc string) error {
	zap.S().Info("Storing context")
	// obscure the password
	ctx.Password = base64.StdEncoding.EncodeToString([]byte(ctx.Password))
	f, err := os.Create(loc)
	if err != nil {
		return err
	}

	defer f.Close()

	encoder := json.NewEncoder(f)
	return encoder.Encode(ctx)
}

// LoadContext returns the information for communication with PF9 controller.
func LoadContext(loc string) (Context, error) {
	zap.S().Info("Loading context...")

	f, err := os.Open(loc)
	if err != nil {

		if os.IsNotExist(err) {
			return Context{}, errors.New("Context absent")
		}
		return Context{}, err
	}

	defer f.Close()

	ctx := Context{WaitPeriod: time.Duration(60), AllowInsecure: false}
	err = json.NewDecoder(f).Decode(&ctx)
	// decode the password
	// Decoding base64 encoded password
	decodedBytePassword, err := base64.StdEncoding.DecodeString(ctx.Password)
	if err != nil {
		return ctx, err
	}
	ctx.Password = string(decodedBytePassword)

	return ctx, err
}
