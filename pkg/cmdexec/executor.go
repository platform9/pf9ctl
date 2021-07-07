// Copyright © 2020 The Platform9 Systems Inc.

package cmdexec

import (
	"fmt"
	"os/exec"

	"github.com/platform9/pf9ctl/pkg/ssh"
	"go.uber.org/zap"
)

const (
	httpsProxy = "https_proxy"
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
	args = append([]string{name}, args...)
	cmd := exec.Command("sudo", args...)
	cmd.Env = append(cmd.Env, httpsProxy+"="+c.ProxyUrl)
	return cmd.Run()
}

// RunWithStdout runs a command locally returning stdout and err
func (c LocalExecutor) RunWithStdout(name string, args ...string) (string, error) {
	args = append([]string{name}, args...)
	cmd := exec.Command("sudo", args...)
	cmd.Env = append(cmd.Env, httpsProxy+"="+c.ProxyUrl)
	byt, err := cmd.Output()
	stderr := ""
	if exitError, ok := err.(*exec.ExitError); ok {
		stderr = string(exitError.Stderr)
	}
	zap.S().Debug("Ran command ", "sudo", args)
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
	zap.S().Debug("Running command ", cmd, "stdout:", string(stdout), "stderr:", string(stderr))
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
