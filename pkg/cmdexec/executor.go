// Copyright © 2020 The Platform9 Systems Inc.
package cmdexec

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/platform9/pf9ctl/pkg/util"

	"github.com/platform9/pf9ctl/pkg/ssh"
	"go.uber.org/zap"
)

// To fetch the stderr after executing command
var StdErrSudoPassword string

const (
	httpsProxy = "https_proxy"
	env_path   = "PATH"
)

// Executor interace abstracts us from local or remote execution
type Executor interface {
	Run(name string, args ...string) error
	RunWithStdout(name string, args ...string) (string, error)
}

// LocalExecutor as the name implies executes commands locally
type LocalExecutor struct {
	ProxyUrl string
}

// Run runs a command locally returning just success or failure
func (c LocalExecutor) Run(name string, args ...string) error {
	if c.ProxyUrl != "" {
		args = append([]string{httpsProxy + "=" + c.ProxyUrl, name}, args...)
	} else {
		args = append([]string{name}, args...)
	}
	cmd := exec.Command("sudo", args...)
	cmd.Env = append(cmd.Env, httpsProxy+"="+c.ProxyUrl)
	cmd.Env = append(cmd.Env, env_path+"="+os.Getenv("PATH"))
	return cmd.Run()
}

// RunWithStdout runs a command locally returning stdout and err
func (c LocalExecutor) RunWithStdout(name string, args ...string) (string, error) {
	if c.ProxyUrl != "" {
		args = append([]string{httpsProxy + "=" + c.ProxyUrl, name}, args...)
	} else {
		args = append([]string{name}, args...)
	}
	cmd := exec.Command("sudo", args...)
	cmd.Env = append(cmd.Env, httpsProxy+"="+c.ProxyUrl)
	byt, err := cmd.Output()
	stderr := ""
	if exitError, ok := err.(*exec.ExitError); ok {
		stderr = string(exitError.Stderr)
	}

	// To append args to a single command
	command := ""
	for _, arg := range args {
		command = fmt.Sprintf("%s \"%s\"", command, arg)
	}
	// Avoid password from getting logged, if the command contains password flag.
	command = PasswordRemover(command)
	zap.S().Debug("Ran command sudo", command)

	zap.S().Debug("stdout:", string(byt), "stderr:", stderr)
	return string(byt), err
}

// RemoteExecutor as the name implies runs commands usign SSH on remote host
type RemoteExecutor struct {
	Client   ssh.Client
	proxyURL string
}

// Run runs a command locally returning just success or failure
func (r *RemoteExecutor) Run(name string, args ...string) error {
	_, err := r.RunWithStdout(name, args...)
	return err
}

// RunWithStdout runs a command locally returning stdout and err
func (r *RemoteExecutor) RunWithStdout(name string, args ...string) (string, error) {
	cmd := name
	for _, arg := range args {
		cmd = fmt.Sprintf("%s \"%s\"", cmd, arg)
	}
	if r.proxyURL != "" {
		cmd = fmt.Sprintf("%s=%s %s", httpsProxy, r.proxyURL, cmd)
	}
	stdout, stderr, err := r.Client.RunCommand(cmd)
	// To fetch the stderr after executing command.
	StdErrSudoPassword = string(stderr)

	// Avoid password from getting logged, if the command contains password flag.
	command := PasswordRemover(cmd)

	zap.S().Debug("Running command ", command, "stdout:", string(stdout), "stderr:", string(stderr))
	return string(stdout), err
}

// NewRemoteExecutor create an Executor interface to execute commands remotely
func NewRemoteExecutor(host string, port int, username string, privateKey []byte, password, proxyURL string) (Executor, error) {
	client, err := ssh.NewClient(host, port, username, privateKey, password, proxyURL)
	if err != nil {
		return nil, err
	}
	re := &RemoteExecutor{Client: client, proxyURL: proxyURL}
	return re, nil
}

// Avoid password from getting logged, if the command contains password flag
func PasswordRemover(cmd string) string {
	// To find the command that contains password flag

	for _, flag := range util.Confidential {
		// If password flag found, then remove the user password.
		if strings.Contains(cmd, flag) {

			index := strings.Index(cmd, flag)
			lastindexbreak := strings.Index(cmd[index:], " ")

			// If password is the last flag in the command.
			if lastindexbreak < 0 {
				cmd = cmd[:index] + flag + "='*****']"
			} else {
				// If password flag is present in the middle or start of the command.
				cmd = cmd[:index] + flag + "='*****'" + cmd[index+lastindexbreak:]
			}
		}
	}
	return cmd
}
