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

type methodGenerator struct {
	outputStyle output
}

func (g methodGenerator) generate(d dex.Data, includeClasses bool, packageFilter string, maxDepth uint, filter filter) countState {
	state := countState{}
	state.packageTree = newNode()

	methodRefs := getMethodRefs(d, filter)

	for _, methodRef := range methodRefs {
		classDescriptor := methodRef.DeclClass
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

func nodeForNamePieces(packageTree *node, namePieces []string, maxDepth uint) {
	for _, pieces := range stringsSequence(namePieces, maxDepth) {
		incrementCount(packageTree, pieces)
	}
}

func incrementCount(n *node, pieces []string) {
	if len(pieces) == 0 {
		n.count++
		return
	}

	name := pieces[0]
	if len(name) == 0 {
		// This method is declared in a class that is part of the default package.
		// Typical examples are methods that operate on arrays of primitive data
		// types.
		name = "<default>"
	}
	child, exists := n.children[name]
	if exists {
		incrementCount(child, pieces[1:])
		return
	}

	newNode := newNode()
	n.names = append(n.names, name)
	n.children[name] = &newNode
	incrementCount(&newNode, pieces[1:])
}

func stringsSequence(strs []string, maxDepth uint) [][]string {
	seq := make([][]string, 0)

	for i := uint(0); i < uint(len(strs))+1 && i < maxDepth; i++ {
		seq = append(seq, strs[:i])
	}

	return seq
}

func getMethodRefs(dexData dex.Data, filter filter) []dex.MethodRef {
	methodRefs := dexData.GetMethodRefs()
	fmt.Println("Read in", len(methodRefs), "method IDs.")
	if filter.val == filterAll {
		return methodRefs
	}

	externalClassRefs := dexData.GetExternalReferences()
	fmt.Println("Read in", len(externalClassRefs), "external class references.")

	externalMethodRefs := map[methodRefKey]struct{}{}
	for _, classRef := range externalClassRefs {
		for _, methodRef := range classRef.MethodRefs {
			externalMethodRefs[newMethodRefKey(methodRef)] = struct{}{}
		}
	}
	fmt.Println("Read in", len(externalMethodRefs), "external method references.")

	filteredMethodRefs := make([]dex.MethodRef, 0)
	for _, methodRef := range methodRefs {
		_, isExternal := externalMethodRefs[newMethodRefKey(methodRef)]
		if (filter.val == filterDefinedOnly && !isExternal) || (filter.val == filterReferencedOnly && isExternal) {
			filteredMethodRefs = append(filteredMethodRefs, methodRef)
		}
	}

	if filter.val == filterDefinedOnly {
		fmt.Println("Filtered to", len(filteredMethodRefs), "defined.")
	} else {
		fmt.Println("Filtered to", len(filteredMethodRefs), "referenced.")
	}

	return filteredMethodRefs
}

type methodRefKey struct {
	// The name of the method's declaring class.
	declClass string
	argTypes  string
	// The method's return type. Examples: "Ljava/lang/String;", "[I".
	returnType string
	methodName string
}

func newMethodRefKey(m dex.MethodRef) methodRefKey {
	return methodRefKey{
		declClass:  m.DeclClass,
		argTypes:   strings.Join(m.ArgTypes, ","),
		returnType: m.ReturnType,
		methodName: m.MethodName,
	}
}
