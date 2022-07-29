package cmdexec

import (
	"testing"

	"github.com/platform9/pf9ctl/pkg/objects"
	"github.com/platform9/pf9ctl/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestGetExecutor(t *testing.T) {
	proxyURL := "127.0.0.1"

	localExecutor := LocalExecutor{ProxyUrl: proxyURL}

	// Test local executor
	executor, err := GetExecutor(proxyURL, util.Node, &objects.NodeConfig{})
	assert.Equal(t, nil, err)
	assert.Equal(t, localExecutor, executor)
}
