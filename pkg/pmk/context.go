package pmk

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/platform9/pf9ctl/pkg/logger"
)

//Context stores information to contact with the
// pf9 controller.
type Context struct {
	Fqdn     string `json:"fqdn"`
	Username string `json:"os_username"`
	Password string `json:"os_password"`
	Tenant   string `json:"os_tenant"`
	Region   string `json:"os_region"`
}

// StoreContext simply updates the in-memory object
func StoreContext(ctx Context, loc string) error {
	logger.Log.Info("Storing context")

	f, err := os.Create(loc)
	if err != nil {
		return err
	}

	defer f.Close()

	encoder := json.NewEncoder(f)
	return encoder.Encode(ctx)
}

// LoadContext returns the information for communication
// with PF9 controller.
func LoadContext(loc string) (Context, error) {
	logger.Log.Info("Loading context...")

	f, err := os.Open(loc)
	if err != nil {

		if os.IsNotExist(err) {
			return Context{}, errors.New("Context absent")
		}
		return Context{}, err
	}

	defer f.Close()

	ctx := Context{}
	err = json.NewDecoder(f).Decode(&ctx)
	return ctx, err
}
