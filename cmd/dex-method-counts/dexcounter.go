/*
Copyright 2017 Rashad Sookram
Copyright Mihai Parparita

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import "github.com/rsookram/dex-method-counts/internal/dex"

type dexCounter struct {
	generator
	countState
	outputStyle output
}

type countState struct {
	overallCount int
	packageTree  node
}

func newDexCounter(countFields bool, outputStyle output) dexCounter {
	return dexCounter{
		generator: newGenerator(countFields, outputStyle),
		countState: countState{
			packageTree: newNode(),
		},
		outputStyle: outputStyle,
	}
}

func (c *dexCounter) generate(d dex.Data, includeClasses bool, packageFilter string, maxDepth uint, filter filter) {
	state := c.generator.generate(d, includeClasses, packageFilter, maxDepth, filter)
	c.countState = mergeCountState(c.countState, state)
}

func mergeCountState(s, s2 countState) countState {
	return countState{
		overallCount: s.overallCount + s2.overallCount,
		packageTree:  *mergeNodes(&s.packageTree, &s2.packageTree),
	}
}

func (c dexCounter) output() {
	c.packageTree.output(c.outputStyle)
}
