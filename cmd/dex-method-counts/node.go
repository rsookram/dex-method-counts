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

import (
	"fmt"
	"sort"
)

type node struct {
	count    int
	names    []string
	children map[string]*node
}

func newNode() node {
	return node{
		children: make(map[string]*node),
	}
}

func mergeNodes(n, n2 *node) *node {
	return &node{
		count:    n.count + n2.count,
		names:    mergeNames(n.names, n2.names),
		children: mergeChildren(n.children, n2.children),
	}
}

func mergeNames(n, n2 []string) []string {
	allNames := make([]string, 0)
	allNames = append(allNames, n...)
	allNames = append(allNames, n2...)

	uniqueNames := make(map[string]struct{})
	for _, name := range allNames {
		uniqueNames[name] = struct{}{}
	}

	mergedNames := make([]string, 0)
	for name, _ := range uniqueNames {
		mergedNames = append(mergedNames, name)
	}
	sort.Strings(mergedNames)

	return mergedNames
}

func mergeChildren(c, c2 map[string]*node) map[string]*node {
	merged := make(map[string]*node)

	for name, n := range c {
		otherNode, otherContains := c2[name]

		if otherContains {
			merged[name] = mergeNodes(n, otherNode)
		} else {
			merged[name] = n
		}
	}

	for name, n := range c2 {
		_, otherContains := c[name]

		// nodes contained in both are handled above
		if !otherContains {
			merged[name] = n
		}
	}

	return merged
}

func (n node) output(style output) {
	if style.val == outputTree {
		n.outputTree("")
	} else if style.val == outputFlat {
		n.outputFlat()
	}
}

func (n node) outputTree(indent string) {
	if len(indent) == 0 {
		fmt.Println("<root>:", n.count)
	}
	indent += "    "

	for _, name := range n.names {
		child := n.children[name]
		fmt.Println(indent+name+":", child.count)
		child.outputTree(indent)
	}
}

func (n node) outputFlat() {
	for _, name := range n.names {
		displayName := name
		if name == "" {
			displayName = "<no package>"
		}
		fmt.Printf("%6d %s\n", n.children[name].count, displayName)
	}
}
