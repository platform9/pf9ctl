// Copyright © 2020 The Platform9 Systems Inc.
package cmdexec

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	"github.com/platform9/pf9ctl/pkg/color"
	"github.com/platform9/pf9ctl/pkg/objects"
	"github.com/platform9/pf9ctl/pkg/ssh"
	"github.com/platform9/pf9ctl/pkg/util"
	"github.com/schollz/progressbar/v3"
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
	RunWithProgressStages(name string, args ...string) (string, error)
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

// RunWithProgressBar runs a command locally displaying the progress status along with stdout and err
func (c LocalExecutor) RunWithProgressStages(name string, args ...string) (string, error) {
	if c.ProxyUrl != "" {
		args = append([]string{httpsProxy + "=" + c.ProxyUrl, name}, args...)
	} else {
		args = append([]string{name}, args...)
	}
	cmd := exec.Command("sudo", args...)
	cmd.Env = append(cmd.Env, httpsProxy+"="+c.ProxyUrl)
	cmd.Env = append(cmd.Env, env_path+"="+os.Getenv("PATH"))

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("unable to pipe stdout: %s", err)
	}
	var stdout string = ""

	stdoutScanner := bufio.NewScanner(stdoutPipe)
	
	// Configurations for progressbar
	bar := progressbar.NewOptions(100,
		progressbar.OptionSetDescription("Downloading Hostagent...	"),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionSetTheme(progressbar.ThemeDefault),
		progressbar.OptionOnCompletion(func() {
			fmt.Println(color.Green("✓ ") + "Hostagent download complete")
		}),
	)

	currentProgress := 0
	nextStageProgress := util.HostAgentprogressPercentage[0]

	if err = cmd.Start(); err != nil {
		fmt.Printf("Unable to start the execution of command: %s\n", err)
		zap.S().Errorf("Error: %s", err.Error())
		return "", fmt.Errorf("unable to pipe stdout: %s", err)
	}

	s := spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	s.Color("red")

	//pre-install checks
	s.Start()
	s.Suffix = "Setting Up Required Components..."
	for stdoutScanner.Scan() {
		outputLine := stdoutScanner.Text()
		stdout += outputLine
		if strings.Contains(outputLine, "distro_install_routine executed successfully") {
			s.Stop()
			fmt.Println(color.Green("✓ ") + "Required components set up successfully")
			break
		}
	}

	//Hostagent Download progressbar incrementation(Fake progress)
	var wg sync.WaitGroup
	quit := make(chan bool)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-quit:
				return
			default:
				if currentProgress < nextStageProgress-1 {
					currentProgress++
					bar.Set(currentProgress)
				}
				time.Sleep(2 * time.Second)
			}
		}
	}()

	//Hostagent Download progressbar incrementation(based on download stages)
	wg.Add(1)
	go func() {
		defer wg.Done()
		stageCount := 0
		for stdoutScanner.Scan() {
			outputLine := stdoutScanner.Text()
			if stageCount < len(util.HostAgentProgressBarStages) && util.HostAgentProgressBarStages[stageCount].MatchString(outputLine) {
				nextStageProgress = util.HostAgentprogressPercentage[stageCount]
				bar.Set(nextStageProgress)
				currentProgress = nextStageProgress
				stageCount++
				if stageCount == len(util.HostAgentProgressBarStages) {
					quit <- true
					break
				} else {
					nextStageProgress = util.HostAgentprogressPercentage[stageCount]
				}
			}
			stdout += outputLine + "\n"
		}

		//Check for Hostagent installation
		s.Start()
		s.Suffix = "Installing Platform9 hostagent..."
		for stdoutScanner.Scan() {
			outputLine := stdoutScanner.Text()
			if strings.Contains(outputLine, "post_install_routine executed successfully") {
				s.Stop()
				fmt.Println(color.Green("✓ ") + "Hostagent installation succeeded")
				break
			}
		}
	}()

	command := ""
	for _, arg := range args {
		command = fmt.Sprintf("%s \"%s\"", command, arg)
	}
	command = ConfidentialInfoRemover(command)
	zap.S().Debug("Ran command sudo", command)

	if err := cmd.Wait(); err != nil {
		zap.S().Debug("stdout:", stdout, "stderr:")
		zap.S().Errorf("Error: %s", err.Error())
		fmt.Println(color.Red("x "), "Package installation failed")
		return stdout, err
	}

	wg.Wait()

	zap.S().Debug("stdout:", stdout, "stderr:")
	return stdout, nil
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

// RunWithProgressBar runs a command remote host displaying the progress status along with stdout
func (r *RemoteExecutor) RunWithProgressStages(name string, args ...string) (string, error) {
	//******to-do******
	//this is a placeholder function, to be implemented
	return "", nil
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

func GetExecutor(proxyURL string, nc objects.NodeConfig) (Executor, error) {
	if CheckRemote(nc) {
		var pKey []byte
		var err error
		if nc.SshKey != "" {
			pKey, err = ioutil.ReadFile(nc.SshKey)
			if err != nil {
				zap.S().Fatalf("Unable to read the sshKey %s, %s", nc.SshKey, err.Error())
			}
		}
		return NewRemoteExecutor(nc.IPs[0], 22, nc.User, pKey, nc.Password, proxyURL)
	}
	zap.S().Debug("Using local executor")
	return LocalExecutor{ProxyUrl: proxyURL}, nil
}

func CheckRemote(nc objects.NodeConfig) bool {
	for _, ip := range nc.IPs {
		if ip != "localhost" && ip != "127.0.0.1" && ip != "::1" {
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
