package myfsm

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func ScanFile(filePath string) ([]Event, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("Файл %s не существает (%s)", filePath, err.Error())
	}

	fileName := filepath.Base(filePath)
	fileNameWithoutExt := "20" + strings.TrimSuffix(fileName, filepath.Ext(fileName))

	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}

	fsm := myFSM{fileName: fileNameWithoutExt}

	defer func() {
		file.Close()
	}()

	scanner := bufio.NewScanner(file)
	scanner.Split(scanLinesWithoutBOM)

	scanner.Buffer(make([]byte, 0), 1024*1024)
	reNewEvent := regexp.MustCompile(`^\d\d:\d\d\.\d{6}`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if reNewEvent.MatchString(line) {
			if fsm.Event != nil {
				fsm.Event = fsm.FinalizeEvent
			} else {
				fsm.Event = fsm.NewEvent
			}
		} else {
			line = "\n" + line
		}

		for _, c := range line {
			fsm.Update(c)
		}

	}
	a, err := file.Seek(0, os.SEEK_CUR)
	func(a int64, err error, s string) {
		func(a int64) {}(a)
	}(a, err, filePath)

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if fsm.Event != nil {
		fsm.Event = fsm.FinalizeEvent
		fsm.Update(0)
	}

	return fsm.events, nil

}
