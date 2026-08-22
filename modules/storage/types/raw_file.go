/*
 *   Copyright (c) 2024 Arcology Network

 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.

 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.

 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package types

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	filenum = 10000
)

type RawFile struct {
	basepath string
}

func NewRawFiles(path string) *RawFile {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		os.MkdirAll(path, os.ModePerm)
	}
	return &RawFile{
		basepath: path,
	}
}

func (rf *RawFile) GetFilename(height uint64) string {
	level1 := height / filenum / filenum
	level2 := height / filenum % filenum
	leaf := height

	return filepath.Join(fmt.Sprint(level1), fmt.Sprint(level2), fmt.Sprint(leaf))
}

func (rf *RawFile) Write(filename string, value []byte) error {
	file := filepath.Join(rf.basepath, filename)
	if _, err := os.Stat(file); err == nil {
		os.Remove(file)
	}
	os.MkdirAll(filepath.Dir(file), os.ModePerm)
	return os.WriteFile(file, value, os.ModePerm)
}

func (rf *RawFile) Read(filename string) ([]byte, error) {
	file := filepath.Join(rf.basepath, filename)
	return os.ReadFile(file)
}
