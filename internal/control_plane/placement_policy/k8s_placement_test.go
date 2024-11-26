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

package placement_policy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLeastAllocated(t *testing.T) {
	// values taken from k8s.io/kubernetes/pkg/scheduler/framework/plugins/noderesources:690
	request := ExtendCPU(CreateResourceMap(1000, 2000))
	n1_occupied := ExtendCPU(CreateResourceMap(2000, 4000))
	n2_occupied := ExtendCPU(CreateResourceMap(1000, 2000))
	n1_capacity := ExtendCPU(CreateResourceMap(4000, 10000))
	n2_capacity := ExtendCPU(CreateResourceMap(6000, 10000))

	n1_score := ScoreFitLeastAllocated(*n1_capacity, *SumResources(request, n1_occupied))
	n2_score := ScoreFitLeastAllocated(*n2_capacity, *SumResources(request, n2_occupied))

	assert.True(t, n1_score == 32 && n2_score == 63, "Least allocated scoring test failed.")
}

func TestBalancedAllocation(t *testing.T) {
	request := ExtendCPU(CreateResourceMap(3000, 5000))
	n1_occupied := ExtendCPU(CreateResourceMap(3000, 0))
	n2_occupied := ExtendCPU(CreateResourceMap(3000, 5000))
	n1_capacity := ExtendCPU(CreateResourceMap(10000, 20000))
	n2_capacity := ExtendCPU(CreateResourceMap(10000, 50000))

	n1_score := ScoreBalancedAllocation(*n1_capacity, *SumResources(request, n1_occupied))
	n2_score := ScoreBalancedAllocation(*n2_capacity, *SumResources(request, n2_occupied))

	assert.True(t, n1_score == 82 && n2_score == 80, "Least allocated scoring test failed.")
}
