/*
Copyright 2017 Rashad Sookram
Copyright (C) 2009 The Android Open Source Project

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

package dex

import (
	"fmt"
	"os"
	"strings"
)

// Converts a single-character primitive type into its human-readable
// equivalent.
func primitiveTypeLabel(typeChar byte) string {
	switch typeChar {
	case 'B':
		return "byte"
	case 'C':
		return "char"
	case 'D':
		return "double"
	case 'F':
		return "float"
	case 'I':
		return "int"
	case 'J':
		return "long"
	case 'S':
		return "short"
	case 'V':
		return "void"
	case 'Z':
		return "boolean"
	default:
		fmt.Fprintf(os.Stderr, "Unexpected class char "+string([]byte{typeChar}))
		return "UNKNOWN"
	}
}

// Converts a type descriptor to human-readable "dotted" form. For example,
// "Ljava/lang/String;" becomes "java.lang.String", and "[I" becomes "int[]".
func DescriptorToDot(descr string) string {
	targetLen := len(descr)
	offset := 0
	arrayDepth := 0

	// strip leading [s; will be added to end */
	for targetLen > 1 && descr[offset] == '[' {
		offset++
		targetLen--
	}
	arrayDepth = offset

	if targetLen == 1 {
		descr = primitiveTypeLabel(descr[offset])
		offset = 0
		targetLen = len(descr)
	} else {
		// account for leading 'L' and trailing ';'
		if targetLen >= 2 && descr[offset] == 'L' && descr[offset+targetLen-1] == ';' {
			targetLen -= 2 // two fewer chars to copy
			offset++       // skip the 'L'
		}
	}

	buf := make([]byte, targetLen+arrayDepth*2)

	// copy class name over
	i := 0
	for i = 0; i < targetLen; i++ {
		ch := descr[offset+i]
		if ch == '/' {
			buf[i] = '.'
		} else {
			buf[i] = ch
		}
	}

	// add the appropriate number of brackets for arrays
	for arrayDepth > 0 {
		arrayDepth--
		buf[i] = '['
		i++
		buf[i] = ']'
		i++
	}

	return string(buf)
}

// Extracts the package name from a type descriptor, and returns it in dotted
// form.
func PackageNameOnly(typeName string) string {
	dotted := DescriptorToDot(typeName)

	end := strings.LastIndexByte(dotted, '.')
	if end < 0 {
		// lives in default package
		return ""
	}

	return dotted[:end]
}
