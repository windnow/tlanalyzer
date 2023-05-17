package common

import (
	"compress/gzip"
	"encoding/json"
	"io"
)

func Compress(compressed io.Writer, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	w := gzip.NewWriter(compressed)
	if _, err := w.Write(payload); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	return nil

}
