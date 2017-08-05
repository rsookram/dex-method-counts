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
	"errors"
	"strings"
)

const (
	filterAll = iota
	filterDefinedOnly
	filterReferencedOnly
)

// Defaults to having val of filterAll
type filter struct {
	val int
}

func (f filter) String() string {
	switch f.val {
	case filterAll:
		return "ALL"
	case filterDefinedOnly:
		return "DEFINED_ONLY"
	case filterReferencedOnly:
		return "REFERENCED_ONLY"
	default:
		return "UNKNOWN"
	}
}

func (f *filter) Set(s string) error {
	s = strings.ToLower(s)

	switch s {
	case "all":
		f.val = filterAll
		return nil
	case "defined_only":
		f.val = filterDefinedOnly
		return nil
	case "referenced_only":
		f.val = filterReferencedOnly
		return nil
	default:
		return errors.New("invalid value " + s)
	}
}
