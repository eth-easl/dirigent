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

func TestDifference(t *testing.T) {
	slice1 := []int{1, 2, 3, 4}
	slice2 := []int{4, 2}

	output := Difference(slice1, slice2)
	assert.Len(t, output, 2)
	assert.Equal(t, output[0], slice1[0])
	assert.Equal(t, output[1], slice1[2])
}
