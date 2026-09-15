package commands

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReportedErrorPreservesWrappedMessageAndIdentity(t *testing.T) {
	cause := errors.New("provider unavailable")
	err := NewReportedError(cause)

	assert.EqualError(t, err, "provider unavailable")
	assert.True(t, errors.Is(err, cause))
	assert.True(t, IsReportedError(err))
	assert.False(t, IsReportedError(cause))
}
