package types

import (
	"fmt"
	"os"
	"strings"
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

	return fmt.Sprintf("%v/%v/%v", level1, level2, leaf)
}

func (rf *RawFile) Write(filename string, value []byte) error {
	file := rf.basepath + "/" + filename
	if _, err := os.Stat(file); err == nil {
		os.Remove(file)
	}
	dirs := strings.Split(filename, "/")
	serchDir := rf.basepath
	for _, dir := range dirs[:len(dirs)-1] {
		serchDir = serchDir + "/" + dir
		_, err := os.Stat(serchDir)
		if os.IsNotExist(err) {
			os.MkdirAll(serchDir, os.ModePerm)
		}

	}
	return os.WriteFile(file, value, os.ModePerm)
}

func (rf *RawFile) Read(filename string) ([]byte, error) {
	file := rf.basepath + "/" + filename
	return os.ReadFile(file)
}
