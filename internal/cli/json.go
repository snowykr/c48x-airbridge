package cli

import (
	"encoding/json"
	"fmt"
	"os"
)

func readJSON[T any](path string) (T, error) {
	var target T
	data, err := os.ReadFile(path)
	if err != nil {
		return target, fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, &target); err != nil {
		return target, fmt.Errorf("parse %s: %w", path, err)
	}
	return target, nil
}
