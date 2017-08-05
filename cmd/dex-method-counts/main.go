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
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/rsookram/dex-method-counts/internal/dex"
)

func main() {
	countFields := flag.Bool("count-fields", false, "")
	includeClasses := flag.Bool("include-classes", false, "")
	packageFilter := flag.String("package-filter", "", "")
	maxDepth := flag.Uint("max-depth", math.MaxUint32, "")

	var filter filter
	flag.Var(&filter, "filter", "")

	var output output
	flag.Var(&output, "output-style", "")

	flag.Parse()

	fileNames := flag.Args()
	if len(fileNames) == 0 {
		fmt.Fprintln(os.Stderr, "No files given")
		os.Exit(1)
	}

	var overallCount int
	for _, fileName := range collectFileNames(fileNames) {
		fmt.Println("Processing " + fileName)

		counter := newDexCounter(*countFields, output)

		dexFiles, err := openInputFiles(fileName)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to open dex files. "+err.Error())
			os.Exit(2)
			return
		}

		for _, f := range dexFiles {
			defer f.Close()
		}

		for _, dexFile := range dexFiles {
			data, err := dex.New(dexFile)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Failed to load dex file "+err.Error())
				os.Exit(2)
			}

			counter.generate(*data, *includeClasses, *packageFilter, *maxDepth, filter)
		}
		counter.output()
		overallCount = counter.overallCount
	}

	fmt.Printf("Overall %s count: %d\n", countFieldsString(*countFields), overallCount)
}

// Opens an input file, which could be a .dex or a .jar/.apk with a classes.dex
// inside. If the latter, we extract the contents to a temporary file.
func openInputFiles(fileName string) ([]os.File, error) {
	dexFiles, err := openInputFileAsZip(fileName)
	if err != nil {
		return []os.File{}, err
	}

	if len(dexFiles) == 0 {
		file, err := os.Open(fileName)
		if err != nil {
			return []os.File{}, err
		}

		return []os.File{*file}, nil
	}

	return dexFiles, err
}

// Tries to open an input file as a Zip archive (jar/apk) with a "classes.dex"
// inside.
func openInputFileAsZip(fileName string) ([]os.File, error) {
	reader, err := zip.OpenReader(fileName)
	if err != nil {
		// Probably not a zip
		return []os.File{}, nil
	}
	defer reader.Close()

	dexFiles := make([]os.File, 0)
	for _, file := range reader.File {
		name := file.Name
		if strings.HasPrefix(name, "classes") && strings.HasSuffix(name, ".dex") {
			dexFile, err := openDexFile(*file)
			if err != nil {
				return []os.File{}, err
			}

			dexFiles = append(dexFiles, *dexFile)
		}
	}

	return dexFiles, nil
}

func openDexFile(zf zip.File) (*os.File, error) {
	fileReader, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer fileReader.Close()

	// Create a temp file to hold the DEX data, open it, and delete it to ensure
	// it doesn't hang around if we fail.
	tmpFile, err := ioutil.TempFile("", "dex")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile.Name())

	// Copy all data from reader to output file.
	if _, err := io.Copy(tmpFile, fileReader); err != nil {
		return nil, err
	}

	return tmpFile, nil
}

func countFieldsString(countFields bool) string {
	if countFields {
		return "field"
	}
	return "method"
}

// Checks if input files array contain directories and adds it's contents to
// the file list if so. Otherwise just adds a file to the list.
func collectFileNames(inputFileNames []string) []string {
	fileNames := make([]string, 0)

	for _, inputFileName := range inputFileNames {
		info, err := os.Stat(inputFileName)
		if err != nil {
			fileNames = append(fileNames, inputFileName)
			continue
		}

		if !info.IsDir() {
			fileNames = append(fileNames, inputFileName)
			continue
		}

		files, err := ioutil.ReadDir(inputFileName)
		if err != nil {
			fileNames = append(fileNames, inputFileName)
			continue
		}

		dirPath := filepath.Clean(inputFileName)
		for _, fileInDir := range files {
			fileNames = append(fileNames, dirPath+string(os.PathSeparator)+fileInDir.Name())
		}
	}

	return fileNames
}
