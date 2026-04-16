package fetcher

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

func FetchText(zipURL string) (string, error) {
	resp, err := http.Get(zipURL)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body failed: %w", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("zip open failed: %w", err)
	}

	for _, f := range zr.File {
		if !strings.HasSuffix(f.Name, ".txt") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", fmt.Errorf("zip entry open failed: %w", err)
		}
		defer rc.Close()

		decoder := japanese.ShiftJIS.NewDecoder()
		reader := transform.NewReader(rc, decoder)
		content, err := io.ReadAll(reader)
		if err != nil {
			return "", fmt.Errorf("decode failed: %w", err)
		}
		return string(content), nil
	}

	return "", fmt.Errorf("no .txt file found in zip")
}
