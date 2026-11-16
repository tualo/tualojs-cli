package compiler

import (
	"strings"
)

func minifyJavaScript(source string) (string, error) {
	tokens, err := tokenize(source)
	if err != nil {
		return "", err
	}
	var output strings.Builder
	var previous *jsToken
	position := 0
	for i := range tokens {
		token := tokens[i]
		gap := compressJavaScriptGap(source[position:token.start])
		output.WriteString(gap)
		if gap == "" && previous != nil && tokensNeedSpace(*previous, token) {
			output.WriteByte(' ')
		}
		output.WriteString(source[token.start:token.end])
		previous = &tokens[i]
		position = token.end
	}
	output.WriteString(compressJavaScriptGap(source[position:]))
	return output.String(), nil
}

func compressJavaScriptGap(gap string) string {
	var output strings.Builder
	for i := 0; i < len(gap); {
		if gap[i] == '/' && i+1 < len(gap) && gap[i+1] == '/' {
			start := i
			for i < len(gap) && gap[i] != '\n' && gap[i] != '\r' {
				i++
			}
			comment := gap[start:i]
			if strings.HasPrefix(strings.TrimSpace(comment), "//!") {
				output.WriteString(comment)
			}
			continue
		}
		if gap[i] == '/' && i+1 < len(gap) && gap[i+1] == '*' {
			start := i
			end := strings.Index(gap[i+2:], "*/")
			if end < 0 {
				output.WriteString(gap[i:])
				break
			}
			i += end + 4
			comment := gap[start:i]
			if strings.HasPrefix(comment, "/*!") {
				output.WriteString(comment)
			} else {
				writeLineEndings(&output, comment)
			}
			continue
		}
		if gap[i] == '\n' || gap[i] == '\r' {
			if gap[i] == '\r' && i+1 < len(gap) && gap[i+1] == '\n' {
				output.WriteString("\r\n")
				i += 2
			} else {
				output.WriteByte(gap[i])
				i++
			}
			continue
		}
		i++
	}
	return output.String()
}

func writeLineEndings(output *strings.Builder, text string) {
	for i := 0; i < len(text); i++ {
		if text[i] == '\r' {
			if i+1 < len(text) && text[i+1] == '\n' {
				output.WriteString("\r\n")
				i++
			} else {
				output.WriteByte('\r')
			}
		} else if text[i] == '\n' {
			output.WriteByte('\n')
		}
	}
}

func tokensNeedSpace(previous, current jsToken) bool {
	if isWordToken(previous) && isWordToken(current) {
		return true
	}
	if previous.kind == numberToken && (current.kind == identifierToken || current.value == ".") {
		return true
	}
	if previous.value == "." && current.kind == numberToken {
		return true
	}
	if previous.kind == punctuationToken && current.kind == punctuationToken {
		combined := previous.value + current.value
		if combined == "//" || combined == "/*" {
			return true
		}
		for _, punctuator := range jsPunctuators {
			if strings.HasPrefix(punctuator, combined) {
				return true
			}
		}
	}
	return false
}

func isWordToken(token jsToken) bool {
	return token.kind == identifierToken || token.kind == numberToken
}
