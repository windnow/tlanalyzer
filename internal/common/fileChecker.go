//go:build windows

package common

import "os"

func FileCanBeDeleted(path string) bool {

	return false

}
func FileExistsAndIsReadable(filename string) bool {
	_, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			// Файл не существует
			return false
		} else {
			// Произошла другая ошибка
			return false
		}
	}
	// Файл существует, проверяем, может ли он быть прочитан
	file, err := os.Open(filename)
	if err != nil {
		// Не удалось прочитать файл
		return false
	}
	defer file.Close()
	return true
}
