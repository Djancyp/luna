package utils

import (
	"fmt"
	"os"
)

func IsFolderExist(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return err
	}
	return nil
}

func IsFileExist(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return err
	}
	dir := !info.IsDir()
	if !dir {
		return fmt.Errorf("file is a directory")
	}
	return nil
}
