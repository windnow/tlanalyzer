package common

import (
	"os"
	"path/filepath"
)

func WorkingDir(wd *string) error {

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	workDir := filepath.Dir(exe)
	*wd, err = filepath.Abs(workDir)
	if err != nil {
		return err
	}

	return nil
}
