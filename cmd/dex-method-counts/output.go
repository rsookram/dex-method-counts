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
	outputTree = iota
	outputFlat
)

// Defaults to having val of outputTree
type output struct {
	val int
}

func (o output) String() string {
	switch o.val {
	case outputTree:
		return "TREE"
	case outputFlat:
		return "FLAT"
	default:
		return "UNKNOWN"
	}
}

func (o *output) Set(s string) error {
	s = strings.ToLower(s)

	switch s {
	case "tree":
		o.val = outputTree
		return nil
	case "flat":
		o.val = outputFlat
		return nil
	default:
		return errors.New("invalid value " + s)
	}
}
