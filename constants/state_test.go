package constants

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStateValue(t *testing.T) {
	assert.Equal(t, State(1), StateEmpty)
	assert.Equal(t, State(2), StatePoll)
	assert.Equal(t, State(3), StateCallback)
	assert.Equal(t, State(4), StateSuccess)
	assert.Equal(t, State(5), StateFail)
}
