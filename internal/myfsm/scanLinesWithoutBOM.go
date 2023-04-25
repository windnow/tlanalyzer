package myfsm

import "bufio"

func scanLinesWithoutBOM(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if len(data) > 0 && data[0] == '\xef' && len(data) > 2 && data[1] == '\xbb' && data[2] == '\xbf' {
		return 3, data[3:], nil
	}
	return bufio.ScanLines(data, atEOF)
}
