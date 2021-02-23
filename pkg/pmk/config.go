package pmk

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"time"

	"go.uber.org/zap"
)

var ErrConfigurationDetailsNotProvided = errors.New("config not set,....")

// Config stores information to contact with the pf9 controller.
type Config struct {
	Fqdn          string        `json:"fqdn"`
	Username      string        `json:"username"`
	Password      string        `json:"password"`
	Tenant        string        `json:"tenant"`
	Region        string        `json:"region"`
	WaitPeriod    time.Duration `json:"wait_period"`
	AllowInsecure bool          `json:"allow_insecure"`
}

// StoreConfig simply updates the in-memory object
func StoreConfig(ctx Config, loc string) error {
	zap.S().Info("Storing configuration details")
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

// LoadConfig returns the information for communication with PF9 controller.
func LoadConfig(loc string) (Config, error) {
	zap.S().Info("Loading configuration details")

	f, err := os.Open(loc)
	if err != nil {

		if os.IsNotExist(err) {
			return Config{}, ErrConfigurationDetailsNotProvided
		}
		return Config{}, err
	}

	defer f.Close()

	ctx := Config{WaitPeriod: time.Duration(60), AllowInsecure: false}
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
