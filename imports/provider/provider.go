package provider

import (
	"errors"
	"github.com/bionic-dev/bionic/internal/provider/database"
	"os"
)

var ErrInputPathShouldBeDirectory = errors.New("input path should be directory")
var ErrInputPathShouldBeFile = errors.New("input path should be file")

type ImportFn struct {
	name      string
	fn        func(inputPath string) error
	inputPath string
}

func NewImportFn(name string, fn func(inputPath string) error, inputPath string) ImportFn {
	return ImportFn{
		name:      name,
		fn:        fn,
		inputPath: inputPath,
	}
}

func (fn ImportFn) Name() string {
	return fn.name
}

func (fn ImportFn) Call() error {
	return fn.fn(fn.inputPath)
}

type Provider interface {
	database.Database
	Name() string
	TablePrefix() string
	Migrate() error
	ImportFns(inputPath string) ([]ImportFn, error)
}

func IsPathDir(inputPath string) bool {
	info, err := os.Stat(inputPath)
	if err != nil {
		return false
	}

	return info.IsDir()
}

func IsPathExists(inputPath string) bool {
	_, err := os.Stat(inputPath)
	return err == nil
}
