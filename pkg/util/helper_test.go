package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringContainsAny(t *testing.T) {

	base := "Ubuntu16"

	actual := StringContainsAny(base, []string{"16", "18"})
	assert.Equal(t, true, actual)

	actual = StringContainsAny(base, []string{"7.5"})
	assert.Equal(t, false, actual)

}
