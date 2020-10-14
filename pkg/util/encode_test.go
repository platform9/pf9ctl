package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncode(t *testing.T) {
	ip := "Hello World"

	expect := "SGVsbG8gV29ybGQ="
	actual := EncodeString(ip)

	assert.Equal(t, expect, actual)
}

func TestDecode(t *testing.T) {

	encoded := "SGVsbG8gV29ybGQ="
	decoded := "Hello World"

	actual := DecodeString(encoded)
	assert.Equal(t, decoded, actual)
}
