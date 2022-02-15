package cmdexec

import (
	"testing"

	"github.com/platform9/pf9ctl/pkg/objects"
	"github.com/stretchr/testify/assert"
)

func TestGetExecutor(t *testing.T) {
	proxyURL := "127.0.0.1"
	noProxy := ""
	localExecutor := LocalExecutor{ProxyUrl: proxyURL}

	// Test local executor
	executor, err := GetExecutor(proxyURL, noProxy, objects.NodeConfig{})
	assert.Equal(t, nil, err)
	assert.Equal(t, localExecutor, executor)
}
