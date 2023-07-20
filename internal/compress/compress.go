package compress

import (
	"bytes"
	"compress/gzip"
	"fmt"
)

func GzipCompress(data []byte) ([]byte, error) {
	var b bytes.Buffer
	w := gzip.NewWriter(&b)

	_, err := w.Write(data)
	if err != nil {
		return nil, fmt.Errorf("failed write data to compress temporary buffer: %w", err)
	}

	err = w.Close()
	if err != nil {
		return nil, fmt.Errorf("failed close compress writer: %w", err)
	}

	return b.Bytes(), nil
}
