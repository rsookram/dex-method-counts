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
	"strings"

	"github.com/rsookram/dex-method-counts/internal/dex"
)

type fieldGenerator struct {
	outputStyle output
}

func (g fieldGenerator) generate(d dex.Data, includeClasses bool, packageFilter string, maxDepth uint, filter filter) countState {
	state := countState{}
	state.packageTree = newNode()

	fieldRefs := getFieldRefs(d, filter)

	for _, fieldRef := range fieldRefs {
		classDescriptor := fieldRef.DeclClass
		var packageName string

		if includeClasses {
			packageName = strings.Replace(dex.DescriptorToDot(classDescriptor), "$", ".", -1)
		} else {
			packageName = dex.PackageNameOnly(classDescriptor)
		}

		if packageFilter != "" && !strings.HasPrefix(packageName, packageFilter) {
			continue
		}

		state.overallCount++

		if g.outputStyle.val == outputTree {
			packageNamePieces := strings.Split(packageName, ".")
			nodeForNamePieces(&state.packageTree, packageNamePieces, maxDepth)
		} else if g.outputStyle.val == outputFlat {
			node, contained := state.packageTree.children[packageName]
			if !contained {
				state.packageTree.names = append(state.packageTree.names, packageName)
				newNode := newNode()
				node = &newNode
				state.packageTree.children[packageName] = node
			}
			node.count++
		}
	}

	return state
}

func getFieldRefs(dexData dex.Data, filter filter) []dex.FieldRef {
	fieldRefs := dexData.GetFieldRefs()
	fmt.Println("Read in", len(fieldRefs), "field IDs.")
	if filter.val == filterAll {
		return fieldRefs
	}

	externalClassRefs := dexData.GetExternalReferences()
	fmt.Println("Read in", len(externalClassRefs), "external class references.")

	externalFieldRefs := map[dex.FieldRef]struct{}{}
	for _, classRef := range externalClassRefs {
		for _, fieldRef := range classRef.FieldRefs {
			externalFieldRefs[fieldRef] = struct{}{}
		}
	}
	fmt.Println("Read in", len(externalFieldRefs), "external field references.")

	filteredFieldRefs := make([]dex.FieldRef, 0)
	for _, fieldRef := range fieldRefs {
		_, isExternal := externalFieldRefs[fieldRef]
		if (filter.val == filterDefinedOnly && !isExternal) || (filter.val == filterReferencedOnly && isExternal) {
			filteredFieldRefs = append(filteredFieldRefs, fieldRef)
		}
	}

	if filter.val == filterDefinedOnly {
		fmt.Println("Filtered to", len(filteredFieldRefs), "defined.")
	} else {
		fmt.Println("Filtered to", len(filteredFieldRefs), "referenced.")
	}

	return filteredFieldRefs
}
