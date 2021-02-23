// Copyright © 2020 The Platform9 Systems Inc.

package cmdexec

import (
	"fmt"
	"os/exec"

	"github.com/platform9/pf9ctl/pkg/ssh"
	"go.uber.org/zap"
)

type ExecutorType string

const (
	Local  ExecutorType = "local"
	Remote ExecutorType = "remote"
)

// Executor interace abstracts us from local or remote execution
type Executor interface {
	Run(name string, args ...string) error
	RunWithStdout(name string, args ...string) (string, error)
}

type ExecutorConfig struct {
	ExecType   ExecutorType
	Host       string
	Port       int
	Username   string
	PrivateKey []byte
	Password   string
	runAsSudo  bool
}

// ExecutorPair is a pair of executors where User Executor runs all commands as passed
// to it, whereas Sudoer attaches "sudo" to each command, hence making the command
// execute as root.
type ExecutorPair struct {
	User   Executor
	Sudoer Executor
}

func NewExecutorPair(config ExecutorConfig) (*ExecutorPair, error) {
	if config.ExecType == Local {
		return &ExecutorPair{
			User:   NewLocalExecutor(false),
			Sudoer: NewLocalExecutor(true),
		}, nil
	}
	user, err := NewRemoteExecutor(config)
	if err != nil {
		return nil, err
	}
	sudoer, _ := NewRemoteExecutor(config)
	if err != nil {
		return nil, err
	}
	return &ExecutorPair{
		User:   user,
		Sudoer: sudoer,
	}, nil
}

// LocalExecutor as the name implies executes commands locally
type LocalExecutor struct {
	sudo bool
}

// NewLocalExecutor creates a new LocalExecutor
func NewLocalExecutor(runAsSudo bool) Executor {
	return LocalExecutor{runAsSudo}
}

// Run runs a command locally returning just success or failure
func (c LocalExecutor) Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

// RunWithStdout runs a command locally returning stdout and err
func (c LocalExecutor) RunWithStdout(name string, args ...string) (string, error) {
	var byt []byte
	var err error
	if c.sudo {
		args = append([]string{name}, args...)
		byt, err = exec.Command("sudo", args...).Output()
		zap.S().Debug("Ran command ", "sudo ", args)
	} else {
		byt, err = exec.Command(name, args...).Output()
		zap.S().Debug("Ran command ", name, args)
	}

	stderr := ""
	if exitError, ok := err.(*exec.ExitError); ok {
		stderr = string(exitError.Stderr)
	}

	zap.S().Debug("stdout:", string(byt), "stderr:", stderr)
	return string(byt), err
}

// RemoteExecutor as the name implies runs commands using SSH on remote host
type RemoteExecutor struct {
	Client ssh.Client
	sudo   bool
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
	if r.sudo {
		cmd = fmt.Sprintf("sudo %s", cmd)
	}
	stdout, stderr, err := r.Client.RunCommand(cmd)
	zap.S().Debug("Running command ", cmd, "stdout:", string(stdout), "stderr:", string(stderr))
	return string(stdout), err
}

// NewRemoteExecutor create an Executor interface to execute commands remotely
func NewRemoteExecutor(config ExecutorConfig) (Executor, error) {
	client, err := ssh.NewClient(config.Host, config.Port, config.Username, config.PrivateKey, config.Password)
	if err != nil {
		return nil, err
	}
	re := &RemoteExecutor{Client: client, sudo: config.runAsSudo}
	return re, nil
}
