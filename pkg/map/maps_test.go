/*
 * MIT License
 *
 * Copyright (c) 2024 EASL
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

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

func TestDifferenceSecond(t *testing.T) {
	s1 := []string{"a", "b", "c"}
	s2 := []string{"b", "c", "d"}

	lr := Difference(s1, s2)
	assert.NotNil(t, lr, "s1 \\ s2 yielded a wrong result.")
	assert.Equal(t, len(lr), 1, "s1 \\ s2 yielded a wrong result.")
	assert.Equal(t, lr[0], "a", "s1 \\ s2 yielded a wrong result.")

	rr := Difference(s2, s1)
	assert.NotNil(t, rr, "s2 \\ s1 yielded a wrong result.")
	assert.Len(t, rr, 1, "s2 \\ s1 yielded a wrong result.")
	assert.Equal(t, rr[0], "d", "s2 \\ s1 yielded a wrong result.")
}
