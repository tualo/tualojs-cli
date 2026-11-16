package compiler

import (
	"fmt"
	"strings"
)

func preprocessSencha(source, mode string) (string, error) {
	if mode != BuildModeDevelopment && mode != BuildModeProduction {
		return "", fmt.Errorf("unsupported build mode %q", mode)
	}
	lines := strings.SplitAfter(source, "\n")
	var output strings.Builder
	depth := 0
	for _, line := range lines {
		lineContent := strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		lineEnding := line[len(lineContent):]
		marker := debugMarker(lineContent)
		switch marker {
		case "open":
			depth++
			if mode == BuildModeDevelopment {
				output.WriteString(line)
			} else {
				output.WriteString(lineEnding)
			}
		case "close":
			if depth == 0 {
				if mode == BuildModeDevelopment {
					output.WriteString(line)
				} else {
					output.WriteString(lineEnding)
				}
				continue
			}
			depth--
			if mode == BuildModeDevelopment {
				output.WriteString(line)
			} else {
				output.WriteString(lineEnding)
			}
		default:
			if mode == BuildModeDevelopment || depth == 0 {
				output.WriteString(line)
			} else {
				output.WriteString(lineEnding)
			}
		}
	}
	if depth != 0 {
		return "", fmt.Errorf("unterminated //<debug> block")
	}
	return output.String(), nil
}

func debugMarker(line string) string {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "//") {
		return ""
	}
	switch strings.TrimSpace(strings.TrimPrefix(line, "//")) {
	case "<debug>":
		return "open"
	case "</debug>":
		return "close"
	default:
		return ""
	}
}
