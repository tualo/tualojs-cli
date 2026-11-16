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
		_, _, bodyOpen, bodyClose, ok, err := functionBounds(tokens, i)
		if err != nil {
			return "", err
		}
		if !ok {
			continue
		}
		edits = append(edits, functionLexicalDeclarations(tokens, bodyOpen, bodyClose)...)
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

func functionLexicalDeclarations(tokens []jsToken, bodyOpen, bodyClose int) []sourceEdit {
	var edits []sourceEdit
	braces, brackets, parentheses := 0, 0, 0
	for i := bodyOpen + 1; i+1 < bodyClose; i++ {
		if braces == 0 && brackets == 0 && parentheses == 0 && isLexicalDeclaration(tokens, i) {
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

func isLexicalDeclaration(tokens []jsToken, index int) bool {
	if index < 0 || index+1 >= len(tokens) || tokens[index].kind != identifierToken {
		return false
	}
	if tokens[index].value != "let" && tokens[index].value != "const" {
		return false
	}
	if index > 0 && is(tokens, index-1, ".") {
		return false
	}
	next := tokens[index+1]
	return next.kind == identifierToken || is(tokens, index+1, "{") || is(tokens, index+1, "[")
}
