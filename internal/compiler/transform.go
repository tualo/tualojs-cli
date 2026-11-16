package compiler

import (
	"sort"
)

type sourceEdit struct {
	start       int
	end         int
	replacement string
}

// transformJavaScript applies narrowly scoped Sencha-compatible normalizations
// that can be made safely from the lightweight token stream.
func transformJavaScript(source string) (string, error) {
	tokens, err := tokenize(source)
	if err != nil {
		return "", err
	}
	var edits []sourceEdit
	for i := range tokens {
		if !is(tokens, i, "function") {
			continue
		}
		paramsOpen, paramsClose, bodyOpen, bodyClose, ok, err := functionBounds(tokens, i)
		if err != nil {
			return "", err
		}
		if !ok {
			continue
		}
		parameters := functionParameters(tokens, paramsOpen, paramsClose)
		if len(parameters) == 0 {
			continue
		}
		edits = append(edits, shadowedLexicalDeclarations(tokens, bodyOpen, bodyClose, parameters)...)
	}
	if len(edits) == 0 {
		return source, nil
	}
	sort.Slice(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
	result := []byte(source)
	for _, edit := range edits {
		result = append(result[:edit.start], append([]byte(edit.replacement), result[edit.end:]...)...)
	}
	return string(result), nil
}

func functionBounds(tokens []jsToken, functionIndex int) (paramsOpen, paramsClose, bodyOpen, bodyClose int, ok bool, err error) {
	index := functionIndex + 1
	if is(tokens, index, "*") {
		index++
	}
	if index < len(tokens) && tokens[index].kind == identifierToken {
		index++
	}
	for index < len(tokens) && !is(tokens, index, "(") && !is(tokens, index, "{") {
		index++
	}
	if !is(tokens, index, "(") {
		return 0, 0, 0, 0, false, nil
	}
	paramsOpen = index
	paramsClose, err = matchingDelimiter(tokens, paramsOpen, "(", ")")
	if err != nil {
		return 0, 0, 0, 0, false, err
	}
	bodyOpen = paramsClose + 1
	for bodyOpen < len(tokens) && !is(tokens, bodyOpen, "{") && !is(tokens, bodyOpen, ";") {
		bodyOpen++
	}
	if !is(tokens, bodyOpen, "{") {
		return 0, 0, 0, 0, false, nil
	}
	bodyClose, err = matchingDelimiter(tokens, bodyOpen, "{", "}")
	if err != nil {
		return 0, 0, 0, 0, false, err
	}
	return paramsOpen, paramsClose, bodyOpen, bodyClose, true, nil
}

func functionParameters(tokens []jsToken, open, close int) map[string]bool {
	parameters := make(map[string]bool)
	segmentStart := open + 1
	braces, brackets, parentheses := 0, 0, 0
	for i := open + 1; i <= close; i++ {
		atEnd := i == close
		atSeparator := !atEnd && is(tokens, i, ",") && braces == 0 && brackets == 0 && parentheses == 0
		if atEnd || atSeparator {
			for j := segmentStart; j < i; j++ {
				if tokens[j].kind == identifierToken {
					parameters[tokens[j].value] = true
					break
				}
				if is(tokens, j, "{") || is(tokens, j, "[") {
					break
				}
			}
			segmentStart = i + 1
		}
		if atEnd || tokens[i].kind != punctuationToken {
			continue
		}
		switch tokens[i].value {
		case "{":
			braces++
		case "}":
			if braces > 0 {
				braces--
			}
		case "[":
			brackets++
		case "]":
			if brackets > 0 {
				brackets--
			}
		case "(":
			parentheses++
		case ")":
			if parentheses > 0 {
				parentheses--
			}
		}
	}
	return parameters
}

func shadowedLexicalDeclarations(tokens []jsToken, bodyOpen, bodyClose int, parameters map[string]bool) []sourceEdit {
	var edits []sourceEdit
	braces, brackets, parentheses := 0, 0, 0
	for i := bodyOpen + 1; i+1 < bodyClose; i++ {
		if braces == 0 && brackets == 0 && parentheses == 0 && tokens[i].kind == identifierToken && (tokens[i].value == "let" || tokens[i].value == "const") && tokens[i+1].kind == identifierToken && parameters[tokens[i+1].value] {
			edits = append(edits, sourceEdit{start: tokens[i].start, end: tokens[i].end, replacement: "var"})
		}
		if tokens[i].kind != punctuationToken {
			continue
		}
		switch tokens[i].value {
		case "{":
			braces++
		case "}":
			if braces > 0 {
				braces--
			}
		case "[":
			brackets++
		case "]":
			if brackets > 0 {
				brackets--
			}
		case "(":
			parentheses++
		case ")":
			if parentheses > 0 {
				parentheses--
			}
		}
	}
	return edits
}
