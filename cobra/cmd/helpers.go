// Copyright © 2015 Steve Francia <spf@spf13.com>.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

var funcMap = template.FuncMap{
	"comment": commentifyString,
}

var projectPath string

var cmdDirs = [...]string{"cmd", "cmds", "command", "commands"}
var goPaths, srcPaths []string

func init() {
	// Initialize goPaths and srcPaths
	envGoPath := os.Getenv("GOPATH")
	if envGoPath == "" {
		er("$GOPATH is not set")
	}

	goPaths = filepath.SplitList(envGoPath)
	srcPaths = make([]string, 0, len(goPaths))
	for _, goPath := range goPaths {
		srcPaths = append(srcPaths, filepath.Join(goPath, "src"))
	}
}

func er(msg interface{}) {
	fmt.Println("Error:", msg)
	os.Exit(1)
}

// isEmpty checks if a given path is empty.
func isEmpty(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		er(err)
	}
	if fi.IsDir() {
		f, err := os.Open(path)
		if err != nil {
			er(err)
		}
		defer f.Close()
		dirs, err := f.Readdirnames(1)
		if err != nil && err != io.EOF {
			er(err)
		}
		return len(dirs) == 0
	}
	return fi.Size() == 0
}

// exists checks if a file or directory exists.
func exists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if !os.IsNotExist(err) {
		er(err)
	}
	return false
}

func executeTemplate(tmplStr string, data interface{}) (string, error) {
	tmpl, err := template.New("").Funcs(funcMap).Parse(tmplStr)
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, data)
	return buf.String(), err
}

func writeStringToFile(path string, s string) error {
	return safeWriteToDisk(path, strings.NewReader(s))
}

// safeWriteToDisk as WriteToDisk but checks to see if file/directory already exists.
func safeWriteToDisk(inpath string, r io.Reader) (err error) {
	dir := filepath.Dir(inpath)
	ospath := filepath.FromSlash(dir)

	if ospath != "" {
		err = os.MkdirAll(ospath, 0777)
		if err != nil {
			return
		}
	}

	if exists(inpath) {
		return fmt.Errorf("%v already exists", inpath)
	}
	if _, err := os.Stat(inpath); err != nil && !os.IsNotExist(err) {
		return err
	}

	file, err := os.Create(inpath)
	if err != nil {
		return
	}
	defer file.Close()

	_, err = io.Copy(file, r)
	return
}

func commentifyString(in string) string {
	var newlines []string
	lines := strings.Split(in, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "//") {
			newlines = append(newlines, line)
		} else {
			if line == "" {
				newlines = append(newlines, "//")
			} else {
				newlines = append(newlines, "// "+line)
			}
		}
	}
	return strings.Join(newlines, "\n")
}
