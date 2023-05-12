package monitor

import "bufio"

func scanLinesWithoutBOM(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if len(data) > 0 && data[0] == '\xef' && len(data) > 2 && data[1] == '\xbb' && data[2] == '\xbf' {
		a, b, c := bufio.ScanLines(data[3:], atEOF)
		return a, b, c
	}
	a, b, c := bufio.ScanLines(data, atEOF)
	return a, b, c
}
