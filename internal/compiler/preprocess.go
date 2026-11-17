package compiler

import (
	"fmt"
	"strings"
)

type toolkitCondition struct {
	name   string
	active bool
}

type featureCondition struct {
	name   string
	active bool
}

func preprocessSencha(source, mode, toolkit string) (string, error) {
	if mode != BuildModeDevelopment && mode != BuildModeProduction {
		return "", fmt.Errorf("unsupported build mode %q", mode)
	}
	lines := strings.SplitAfter(source, "\n")
	var output strings.Builder
	debugDepth := 0
	var toolkitConditions []toolkitCondition
	var featureConditions []featureCondition
	for _, line := range lines {
		lineContent := strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		lineEnding := line[len(lineContent):]
		toolkitMarker, condition := toolkitConditionMarker(lineContent)
		toolkitActive := len(toolkitConditions) == 0 || toolkitConditions[len(toolkitConditions)-1].active
		featureMarker, feature := featureConditionMarker(lineContent)
		featureActive := len(featureConditions) == 0 || featureConditions[len(featureConditions)-1].active
		active := toolkitActive && featureActive && (debugDepth == 0 || mode == BuildModeDevelopment)

		if toolkitMarker == "open" {
			conditionActive, err := evaluateToolkitCondition(condition, toolkit)
			if err != nil {
				return "", fmt.Errorf("Sencha toolkit condition %q: %w", condition, err)
			}
			toolkitConditions = append(toolkitConditions, toolkitCondition{name: condition, active: toolkitActive && conditionActive})
			output.WriteString(lineEnding)
			continue
		}
		if toolkitMarker == "close" {
			if len(toolkitConditions) == 0 {
				return "", fmt.Errorf("unmatched Sencha toolkit condition close marker")
			}
			current := toolkitConditions[len(toolkitConditions)-1]
			if condition != "" && condition != current.name {
				return "", fmt.Errorf("Sencha toolkit condition close %q does not match open condition %q", condition, current.name)
			}
			toolkitConditions = toolkitConditions[:len(toolkitConditions)-1]
			output.WriteString(lineEnding)
			continue
		}
		if featureMarker == "open" {
			featureEnabled, err := evaluateSenchaFeature(feature)
			if err != nil {
				return "", fmt.Errorf("Sencha feature %q: %w", feature, err)
			}
			featureConditions = append(featureConditions, featureCondition{name: feature, active: featureActive && featureEnabled})
			output.WriteString(lineEnding)
			continue
		}
		if featureMarker == "close" {
			if len(featureConditions) == 0 {
				return "", fmt.Errorf("unmatched Sencha feature close marker")
			}
			featureConditions = featureConditions[:len(featureConditions)-1]
			output.WriteString(lineEnding)
			continue
		}

		marker := debugMarker(lineContent)
		switch marker {
		case "open":
			debugDepth++
			if active && mode == BuildModeDevelopment {
				output.WriteString(line)
			} else {
				output.WriteString(lineEnding)
			}
		case "close":
			if debugDepth == 0 {
				if toolkitActive && featureActive && mode == BuildModeDevelopment {
					output.WriteString(line)
				} else {
					output.WriteString(lineEnding)
				}
				continue
			}
			debugDepth--
			if toolkitActive && featureActive && mode == BuildModeDevelopment {
				output.WriteString(line)
			} else {
				output.WriteString(lineEnding)
			}
		default:
			if active {
				output.WriteString(line)
			} else {
				output.WriteString(lineEnding)
			}
		}
	}
	if debugDepth != 0 {
		return "", fmt.Errorf("unterminated //<debug> block")
	}
	if len(toolkitConditions) != 0 {
		return "", fmt.Errorf("unterminated Sencha toolkit condition block %q", toolkitConditions[len(toolkitConditions)-1].name)
	}
	if len(featureConditions) != 0 {
		return "", fmt.Errorf("unterminated Sencha feature block %q", featureConditions[len(featureConditions)-1].name)
	}
	return output.String(), nil
}

func evaluateToolkitCondition(condition, toolkit string) (bool, error) {
	switch condition {
	case "classic", "modern":
		if toolkit == "" {
			return false, fmt.Errorf("condition %q requires a selected toolkit", condition)
		}
		return toolkit == condition, nil
	case "nonBrowser", "legacyBrowser":
		// This compiler currently builds browser applications for modern browsers.
		return false, nil
	default:
		return false, fmt.Errorf("unknown condition %q", condition)
	}
}

func evaluateSenchaFeature(feature string) (bool, error) {
	switch {
	case feature == "legacyBrowser":
		return false, nil
	case feature == "logger", strings.HasPrefix(feature, "classSystem."):
		return true, nil
	default:
		return false, fmt.Errorf("unknown feature")
	}
}

func toolkitConditionMarker(line string) (string, string) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "//") {
		return "", ""
	}
	marker := strings.TrimSpace(strings.TrimPrefix(line, "//"))
	if strings.HasPrefix(marker, "</if") && strings.HasSuffix(marker, ">") {
		condition := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(marker, "</if"), ">"))
		return "close", condition
	}
	if strings.HasPrefix(marker, "<if ") && strings.HasSuffix(marker, ">") {
		return "open", strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(marker, "<if "), ">"))
	}
	return "", ""
}

func featureConditionMarker(line string) (string, string) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "//") {
		return "", ""
	}
	marker := strings.TrimSpace(strings.TrimPrefix(line, "//"))
	if marker == "</feature>" {
		return "close", ""
	}
	if strings.HasPrefix(marker, "<feature ") && strings.HasSuffix(marker, ">") {
		return "open", strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(marker, "<feature "), ">"))
	}
	return "", ""
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
