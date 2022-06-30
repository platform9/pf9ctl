// Copyright © 2020 The Platform9 Systems Inc.
package cmdexec

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/platform9/pf9ctl/pkg/objects"
	"github.com/platform9/pf9ctl/pkg/ssh"
	"github.com/platform9/pf9ctl/pkg/util"
	"go.uber.org/zap"
)

// To fetch the stderr after executing command
var StdErrSudoPassword string

const (
	httpsProxy = "https_proxy"
	env_path   = "PATH"
)

// Executor interface abstracts us from local or remote execution
type Executor interface {
	Run(name string, args ...string) error
	RunWithStdout(name string, args ...string) (string, error)
	RunCommandWait(command string) string
}

// LocalExecutor as the name implies executes commands locally
type LocalExecutor struct {
	ProxyUrl string
}

func (c LocalExecutor) RunCommandWait(command string) string {
	command = "sudo " + command
	output := exec.Command("/bin/sh", "-c", command)
	output.Stdout = os.Stdout
	output.Stdin = os.Stdin
	err := output.Start()
	output.Wait()
	if err != nil {
		fmt.Println(err.Error())
	}
	return ""
}

func (r *RemoteExecutor) RunCommandWait(command string) string {
	o, err := r.RunWithStdout(command)
	if err != nil {
		zap.S().Debugf("Error :", err.Error())
	}
	return o
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
	cmd.Env = append(cmd.Env, env_path+"="+os.Getenv("PATH"))
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

	// Avoid confidential info in the command from getting logged
	command = ConfidentialInfoRemover(command)
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

	// Avoid confidential info in the command from getting logged
	command := ConfidentialInfoRemover(cmd)

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

// Avoid confidential information from getting logged
func ConfidentialInfoRemover(cmd string) string {
	// To find the command that contains confidential info

	for _, flag := range util.Confidential {
		// If confidential flags are found, remove those
		if strings.Contains(cmd, flag) {

			index := strings.Index(cmd, flag)
			lastindexbreak := strings.Index(cmd[index:], " ")

			// If confidential parameter is the last flag in the command.
			if lastindexbreak < 0 {
				cmd = cmd[:index] + flag + "='*****']"
			} else {
				// If confidential parameter flag is present in the middle or start of the command.
				cmd = cmd[:index] + flag + "='*****'" + cmd[index+lastindexbreak:]
			}
		}
	}
	return cmd
}

func GetExecutor(proxyURL string, nc *objects.NodeConfig) (Executor, error) {
	if CheckRemote(nc) {
		var pKey []byte
		var err error
		if nc.SshKey != "" {
			pKey, err = ioutil.ReadFile(nc.SshKey)
			if err != nil {
				zap.S().Fatalf("Unable to read the sshKey %s, %s", nc.SshKey, err.Error())
			}
		}
		return NewRemoteExecutor(nc.Spec.Nodes[0].Ip, 22, nc.Spec.Nodes[0].Hostname, pKey, nc.Password, proxyURL)
	}
	zap.S().Debug("Using local executor")
	return LocalExecutor{ProxyUrl: proxyURL}, nil
}

func CheckRemote(nc *objects.NodeConfig) bool {
	for _, node := range nc.Spec.Nodes {
		if node.Ip != "localhost" && node.Ip != "127.0.0.1" && node.Ip != "::1" {
			return true
		}
	}
	return false
}

func ExitCodeChecker(err error) (string, int) {
	var stderr string
	var exitCode int
	if exitError, ok := err.(*exec.ExitError); ok {
		stderr = string(exitError.Stderr)
		exitCode = exitError.ExitCode()
	}
	return stderr, exitCode
}
