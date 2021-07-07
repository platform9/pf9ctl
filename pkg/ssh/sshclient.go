// Copyright 2020 Platform9 Systems Inc.
package ssh

// The content of this files are shamelessly copied from the SSH Provider code base of cctl
// the CCTL ssh-provider can't handle large files and hence this step was taken, perhaps
// the original source should have been modified.

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/pkg/sftp"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

// Client interface provides ways to run command and upload files to remote hosts
type Client interface {
	// RunCommand executes the remote command returning the stdout, stderr and any error associated with it
	RunCommand(cmd string) ([]byte, []byte, error)
	// Uploadfile uploads the srcFile to remoteDestFilePath and changes the mode to the filemode
	UploadFile(srcFilePath, remoteDstFilePath string, mode os.FileMode, cb func(read int64, total int64)) error
	// Downloadfile downloads the remoteFile to localFile and changes the mode to the filemode
	DownloadFile(remoteFile, localPath string, mode os.FileMode, cb func(read int64, total int64)) error
}

type client struct {
	sshClient  *ssh.Client
	sftpClient *sftp.Client
	proxyURL   string
}

var (
	SudoPassword string
)

const (
	runAsSudo = true
)

// NewClient creates a new Client that can be used to perform action on a
// machine
func NewClient(host string, port int, username string, privateKey []byte, password, proxyURL string) (Client, error) {

	authMethods := make([]ssh.AuthMethod, 1)
	// give preferece to privateKey
	if privateKey != nil {
		signer, err := ssh.ParsePrivateKey([]byte(privateKey))
		if err != nil {
			return nil, fmt.Errorf("error parsing private key: %s", err)
		}
		authMethods[0] = ssh.PublicKeys(signer)
	} else {
		authMethods[0] = ssh.Password(password)
	}
	sshConfig := &ssh.ClientConfig{
		User: string(username),
		Auth: authMethods,
		// by default ignore host key checks
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	sshClient, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), sshConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to dial %s:%d: %s", host, port, err)
	}
	sftpClient, err := sftp.NewClient(sshClient)
	return &client{
		sshClient:  sshClient,
		sftpClient: sftpClient,
		proxyURL:   proxyURL,
	}, nil
}

// RunCommand runs a command on the machine and returns stdout and stderr
// separately
func (c *client) RunCommand(cmd string) ([]byte, []byte, error) {

	session, err := c.sshClient.NewSession()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create session: %s", err)
	}
	stdOutPipe, err := session.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to pipe stdout: %s", err)
	}
	stdErrPipe, err := session.StderrPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to pipe stderr: %s", err)
	}
	// Prepend sudo if runAsSudo set to true
	if runAsSudo {
		// Prepend Sudo and add if Password is required to access Sudo
		if SudoPassword != "" {
			cmd = fmt.Sprintf("echo %s | sudo -S su ; sudo %s", SudoPassword, cmd)
		} else {
			cmd = fmt.Sprintf("sudo %s", cmd)
		}
	}
	if c.proxyURL != "" {
		cmd = fmt.Sprintf("https_proxy=%s %s", c.proxyURL, cmd)
	}
	err = session.Start(cmd)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to run command: %s", err)
	}
	stdOut, err := ioutil.ReadAll(stdOutPipe)
	stdErr, err := ioutil.ReadAll(stdErrPipe)
	err = session.Wait()
	if err != nil {
		retError := err
		switch err.(type) {
		case *ssh.ExitError:
			retError = fmt.Errorf("command %s failed: %s", cmd, err)
		case *ssh.ExitMissingError:
			retError = fmt.Errorf("command %s failed (no exit status): %s", cmd, err)
		default:
			retError = fmt.Errorf("command %s failed: %s", cmd, err)
		}

		zap.L().Debug("Error ", zap.String("stdout", string(stdOut)), zap.String("stderr", string(stdErr)))

		return stdOut, stdErr, retError
	}
	return stdOut, stdErr, nil
}

// Upload writes a file to the machine
func (c *client) UploadFile(localFile string, remoteFilePath string, mode os.FileMode, cb func(read int64, total int64)) error {
	// first check if the local file exists or not
	localFp, err := os.Open(localFile)
	if err != nil {
		return fmt.Errorf("unable to read localFile: %s", err)
	}
	defer localFp.Close()
	fInfo, err := localFp.Stat()
	if err != nil {
		return fmt.Errorf("Unable to find size of the file %s", localFile)
	}

	localFileReader := bufio.NewReader(localFp)
	// create a progrssReader that will call the callback function after each read
	progressReader := newProgressCBReader(fInfo.Size(), localFileReader, cb)

	remoteFile, err := c.sftpClient.Create(remoteFilePath)
	if err != nil {
		return fmt.Errorf("unable to create file: %s", err)
	}
	defer remoteFile.Close()
	// IMHO this function is misnomer, it actually writes to the remoteFile
	_, err = remoteFile.ReadFrom(progressReader)
	if err != nil {
		// rmove the remote file since write failed and ignore the errors
		// we can't do much about it anyways.
		c.sftpClient.Remove(remoteFilePath)
		return fmt.Errorf("write failed: %s, ", err)
	}
	err = remoteFile.Chmod(mode)
	if err != nil {
		return fmt.Errorf("chmod failed: %s", err)
	}
	return nil
}

// DownloadFile fetches a file from the remote machine
func (c *client) DownloadFile(remoteFile string, localFilePath string, mode os.FileMode, cb func(read int64, total int64)) error {
	// check if remote file exists
	remoteFP, err := c.sftpClient.Open(remoteFile)
	if err != nil {
		return fmt.Errorf("unable to read remoteFile: %s", err)
	}
	defer remoteFP.Close()
	fInfo, err := remoteFP.Stat()
	if err != nil {
		return fmt.Errorf("unable to find size of remoteFile: %s", err)
	}

	remoteFileReader := bufio.NewReader(remoteFP)
	progressReader := newProgressCBReader(fInfo.Size(), remoteFileReader, cb)

	localFile, err := os.Create(localFilePath)
	if err != nil {
		return fmt.Errorf("unable to create local file: %s", err)
	}
	defer localFile.Close()

	_, err = io.Copy(localFile, progressReader)
	if err != nil {
		os.Remove(localFilePath)
		return fmt.Errorf("unable to copy data: %s", err)
	}
	err = localFile.Chmod(mode)
	if err != nil {
		os.Remove(localFilePath)
		return fmt.Errorf("chmod failed: %s", err)
	}
	return nil
}

func newProgressCBReader(totalSize int64, orig io.Reader, cb func(read int64, total int64)) io.Reader {
	progReader := &ProgressCBReader{
		TotalSize:  totalSize,
		ReadCount:  0,
		ProgressCB: cb,
		OrigReader: orig,
	}
	return progReader
}

// ProgressCBReader implements a reader that can call back
// a function on  regular interval to report progress
type ProgressCBReader struct {
	TotalSize  int64
	ReadCount  int64
	ProgressCB func(read int64, total int64)
	OrigReader io.Reader
}

func (r *ProgressCBReader) Read(p []byte) (int, error) {
	read, err := r.OrigReader.Read(p)
	r.ReadCount = r.ReadCount + int64(read)
	if r.ProgressCB != nil {
		r.ProgressCB(r.ReadCount, r.TotalSize)
	}
	return read, err
}
