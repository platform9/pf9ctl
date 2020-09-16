package cmd

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/erwinvaneyk/cobras"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/platform9/pf9ctl/pkg/auth"
)

type LoginOptions struct {
	KeystoneAddr  string
	Insecure      bool
	TokenCacheDir string

	httpClient *http.Client
}

func NewCmdLogin() *cobra.Command {
	opts := &LoginOptions{
		TokenCacheDir: path.Join(os.Getenv("HOME"), ".kube/pf9/cache"),
	}

	cmd := &cobra.Command{
		Use:     "login",
		Short:   "Generate a .",
		Run:     cobras.Run(opts),
	}

	// TODO option to debug
	// TODO allow env vars
	// TODO option to only print token

	cmd.Flags().StringVar(&opts.TokenCacheDir, "cache-dir", opts.TokenCacheDir, "Directory to store the cached authentication token(s),")
	cmd.Flags().StringVar(&opts.KeystoneAddr, "keystone-addr", "", "Endpoint of Keystone. Format: https://some-du.platform9.horse/keystone/v3")
	cmd.Flags().BoolVar(&opts.Insecure, "insecure-skip-tls-verify", false, "If true, the certificate of the authentication server will not be checked for validity. This will make your HTTPS connection insecure.")

	return cmd
}

func (o *LoginOptions) Complete(cmd *cobra.Command, args []string) error {
	logger, err := zap.NewDevelopment()
	if err != nil {
		return err
	}
	zap.ReplaceGlobals(logger)

	if o.Insecure {
		o.httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	}

	return nil
}

func (o *LoginOptions) Validate() error {
	if o.KeystoneAddr == "" {
		return errors.New("keystone address not provided")
	}
	return nil
}

// TODO ensure newline is printed after error
// TODO add name of tool to error
// TODO fix logging
// TODO split out caching logic
// TODO move out complex logic for kubectl/kubeconfig
func (o *LoginOptions) Run(ctx context.Context) error {
	keystoneClient := auth.NewKeystoneClient(o.KeystoneAddr, o.httpClient)

	// Check cache for valid token
	err := os.MkdirAll(o.TokenCacheDir, 0755)
	if err != nil {
		return err
	}

	tokenPath := path.Join(o.TokenCacheDir, "token")

	var tokenInfo auth.TokenInfo
	bs, err := ioutil.ReadFile(tokenPath)
	if err == nil {
		err = json.Unmarshal(bs, &tokenInfo)
		if err != nil {
			return err
		}

		if time.Now().After(tokenInfo.ExpiresAt) {
			return errors.New("token has expired")
		}
	} else {
		// No cached token found, request a new one
		credentials, err := requireCredentials()
		if err != nil {
			return err
		}

		if credentials.Password == "" || credentials.Username == "" {
			return errors.New("invalid username or password")
		}

		fmt.Fprintf(os.Stderr, "%+v\n", credentials)
		tokenInfo, err = keystoneClient.Auth(credentials)
		if err != nil {
			return err
		}

		bs, err := json.Marshal(tokenInfo)
		if err != nil {
			return err
		}


		err = ioutil.WriteFile(tokenPath, bs, 0600)
		if err != nil {
			return err
		}
	}

	return auth.PrintTokenForKubectl(tokenInfo)
}

func requireCredentials() (auth.Credentials, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Fprintln(os.Stderr, "Please login with your Platform9 account.")
	fmt.Fprint(os.Stderr, "Username: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		return auth.Credentials{}, err
	}

	fmt.Fprint(os.Stderr, "Password: ")
	bytePassword, err := terminal.ReadPassword(syscall.Stdin)
	if err != nil {
		return auth.Credentials{}, err
	}
	fmt.Fprint(os.Stderr, "\n")

	return auth.Credentials{
		Username: strings.TrimSpace(username),
		Password: strings.TrimSpace(string(bytePassword)),
	}, nil
}
