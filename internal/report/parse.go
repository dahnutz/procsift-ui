package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// ParseJSON validates and decodes one ProcSift report from JSON bytes.
func ParseJSON(data []byte) (Report, error) {
	if len(data) > MaxBytes {
		return Report{}, fmt.Errorf("report exceeds the 64 MiB input limit")
	}
	var rep Report
	if err := json.Unmarshal(data, &rep); err != nil {
		return Report{}, fmt.Errorf("parse report JSON: %w", err)
	}
	if err := Validate(rep); err != nil {
		return Report{}, err
	}
	return rep, nil
}

// ParseReader reads up to MaxBytes+1 from r and parses a report.
func ParseReader(r io.Reader) (Report, error) {
	b, err := io.ReadAll(io.LimitReader(r, MaxBytes+1))
	if err != nil {
		return Report{}, err
	}
	return ParseJSON(b)
}

// Validate checks tool and version fields on a decoded report.
func Validate(rep Report) error {
	tool := strings.TrimSpace(rep.Tool)
	if tool != "procsift" {
		return fmt.Errorf("report tool %q is not procsift", rep.Tool)
	}
	if rep.Version == "" {
		return fmt.Errorf("report has no version")
	}
	if strings.HasPrefix(rep.Version, "0.5.") || rep.Version == "0.6.0-dev" {
		return nil
	}
	return fmt.Errorf("unsupported report version %q (supported: 0.5.x; 0.6.0-dev testing)", rep.Version)
}
