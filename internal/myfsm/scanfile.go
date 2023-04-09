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

func ScanFile(filePath string, multiplier int) ([]*Event, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("Файл %s не существает (%s)", filePath, err.Error())
	}

	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	fileName := filepath.Base(filePath)
	fileNameWithoutExt := "20" + strings.TrimSuffix(fileName, filepath.Ext(fileName))
	fsm := myFSM{fileName: fileNameWithoutExt}
	scanner := bufio.NewScanner(file)
	if multiplier > 0 {
		scanner.Buffer(make([]byte, 1024*multiplier), bufio.MaxScanTokenSize)
	}
	reNewEvent := regexp.MustCompile(`^\d\d:\d\d\.\d{6}`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) > 2 && "\uFEFF" == string(line[0:3]) {
			line = string(line[3:])
		}
		if reNewEvent.MatchString(line) {
			if fsm.Event != nil {
				fsm.Event = fsm.FinalizeEvent
			} else {
				fsm.Event = fsm.NewEvent
			}
		} else {
			line = "\n" + line
		}

		var c rune

		for _, c = range line {
			fsm.Update(c)
		}
		c = 0

	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if fsm.Event != nil {
		fsm.Event = fsm.FinalizeEvent
		fsm.Update(0)
	}

	return fsm.events, nil

}
