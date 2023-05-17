package common

import "path/filepath"

func GetParentDirectoryName(path string) (string, error) {
	dirPath := filepath.Dir(path)
	dirName := filepath.Base(dirPath)
	return dirName, nil
}
