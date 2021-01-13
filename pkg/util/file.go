package util

import (
	"fmt"
	"io/ioutil"
	"os"

	"go.uber.org/zap"
)

// FileExists returns whether the given file or directory exists
func FileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("File not found: %s", err)
}

// ReadFile tries to read a file and returns raw file data on success
func ReadFile(path string) ([]byte, error) {
	if _, err := FileExists(path); err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err = file.Close(); err != nil {
			zap.S().Error(err)
		}
	}()

	return ioutil.ReadAll(file)
}
