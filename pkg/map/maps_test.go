package _map

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeysValues(t *testing.T) {
	mp := make(map[string]string)
	mp["paris"] = "france"
	mp["london"] = "uk"

	assert.Len(t, Keys(mp), 2)
	assert.Len(t, Values(mp), 2)

	mp["roma"] = "italy"
	mp["bern"] = "switzerland"

	assert.Len(t, Keys(mp), 4)
	assert.Len(t, Values(mp), 4)
}
