/*
 * Copyright (c) 2015 Alex Yatskov <alex@foosoft.net>
 * Author: Alex Yatskov <alex@foosoft.net>
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy of
 * this software and associated documentation files (the "Software"), to deal in
 * the Software without restriction, including without limitation the rights to
 * use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
 * the Software, and to permit persons to whom the Software is furnished to do so,
 * subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
 * FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
 * COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
 * IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
 * CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 */

package goldsmith

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type fileInfo struct {
	os.FileInfo
	path string
}

func cleanPath(path string) string {
	if filepath.IsAbs(path) {
		var err error
		if path, err = filepath.Rel("/", path); err != nil {
			panic(err)
		}
	}

	return filepath.Clean(path)
}

func scanDir(root string, infos chan fileInfo) {
	defer close(infos)

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		infos <- fileInfo{FileInfo: info, path: path}
		return nil
	})
}

func setDelimValue(container interface{}, path string, data interface{}) bool {
	containerVal := reflect.Indirect(reflect.ValueOf(container))

	segments := strings.Split(path, ".")
	segmentHead := segments[0]

	if len(segments) > 1 {
		var fieldVal reflect.Value
		switch containerVal.Kind() {
		case reflect.Map:
			fieldVal = containerVal.MapIndex(reflect.ValueOf(segmentHead))
		case reflect.Struct:
			fieldVal = containerVal.FieldByName(segmentHead)
			if fieldVal.CanAddr() {
				fieldVal = fieldVal.Addr()
			}
		}

		if fieldVal.IsValid() && fieldVal.CanInterface() {
			pathRest := strings.Join(segments[1:], ".")
			return setDelimValue(fieldVal.Interface(), pathRest, data)
		}
	} else {
		switch containerVal.Kind() {
		case reflect.Map:
			containerVal.SetMapIndex(reflect.ValueOf(segmentHead), reflect.ValueOf(data))
			return true
		case reflect.Struct:
			fieldVal := containerVal.FieldByName(segmentHead)
			if fieldVal.CanSet() {
				fieldVal.Set(reflect.ValueOf(data))
				return true
			}
		}
	}

	return false
}

func getDelimValue(container interface{}, path string) (interface{}, bool) {
	containerVal := reflect.Indirect(reflect.ValueOf(container))

	segments := strings.Split(path, ".")
	segmentHead := segments[0]

	var fieldVal reflect.Value
	switch containerVal.Kind() {
	case reflect.Map:
		fieldVal = containerVal.MapIndex(reflect.ValueOf(segmentHead))
	case reflect.Struct:
		fieldVal = containerVal.FieldByName(segmentHead)
	}

	if fieldVal.IsValid() && fieldVal.CanInterface() {
		if len(segments) > 1 {
			return getDelimValue(fieldVal.Interface(), strings.Join(segments[1:], "."))
		}

		return fieldVal.Interface(), true
	}

	return nil, false
}
