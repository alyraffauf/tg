package cli

import (
	"fmt"
	"os"
)

func commandBody(body, bodyFile string) (string, error) {
	if bodyFile == "" {
		return body, nil
	}
	if body != "" {
		return "", fmt.Errorf("--body and --body-file cannot be used together")
	}
	data, err := os.ReadFile(bodyFile)
	if err != nil {
		return "", fmt.Errorf("read body file: %w", err)
	}
	return string(data), nil
}
