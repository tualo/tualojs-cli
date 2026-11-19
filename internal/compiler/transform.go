package compiler

import (
	"sort"
	"strings"
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

// upgradeES5JavaScript converts a conservative subset of function-local var
// declarations to lexical declarations. Anything whose binding or execution
// semantics cannot be established from the token stream is left unchanged.
func upgradeES5JavaScript(source string) (string, error) {
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
		if !ok || functionHasDynamicScope(tokens, bodyOpen, bodyClose) {
			continue
		}
		edits = append(edits, functionVarUpgradeEdits(source, tokens, paramsOpen, paramsClose, bodyOpen, bodyClose)...)
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

func functionHasDynamicScope(tokens []jsToken, bodyOpen, bodyClose int) bool {
	for i := bodyOpen + 1; i < bodyClose; i++ {
		if is(tokens, i, "with") || is(tokens, i, "eval") && is(tokens, i+1, "(") || tokens[i].value == "=>" {
			return true
		}
	}
	return false
}

func functionVarUpgradeEdits(source string, tokens []jsToken, paramsOpen, paramsClose, bodyOpen, bodyClose int) []sourceEdit {
	var edits []sourceEdit
	braces, brackets, parentheses := 0, 0, 0
	for i := bodyOpen + 1; i < bodyClose; i++ {
		if braces == 0 && brackets == 0 && parentheses == 0 && is(tokens, i, "var") {
			if edit, ok := functionVarUpgrade(source, tokens, paramsOpen, paramsClose, bodyOpen, bodyClose, i); ok {
				edits = append(edits, edit)
			}
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

func functionVarUpgrade(source string, tokens []jsToken, paramsOpen, paramsClose, bodyOpen, bodyClose, declaration int) (sourceEdit, bool) {
	if declaration+3 >= bodyClose || tokens[declaration+1].kind != identifierToken || !is(tokens, declaration+2, "=") {
		return sourceEdit{}, false
	}
	// Only ordinary statement-position declarations are safe to replace with
	// lexical declarations. In particular this excludes single-statement ifs.
	if declaration > bodyOpen+1 && !is(tokens, declaration-1, ";") && !is(tokens, declaration-1, "}") {
		return sourceEdit{}, false
	}
	name := tokens[declaration+1].value
	if isUnsafeLexicalBindingName(name) || functionParameterNamed(tokens, paramsOpen, paramsClose, name) {
		return sourceEdit{}, false
	}

	statementEnd := -1
	depthBraces, depthBrackets, depthParens := 0, 0, 0
	for i := declaration + 3; i < bodyClose; i++ {
		if tokens[i].kind == punctuationToken {
			switch tokens[i].value {
			case "{":
				depthBraces++
			case "}":
				if depthBraces == 0 && depthBrackets == 0 && depthParens == 0 {
					return sourceEdit{}, false
				}
				if depthBraces > 0 {
					depthBraces--
				}
			case "[":
				depthBrackets++
			case "]":
				if depthBrackets > 0 {
					depthBrackets--
				}
			case "(":
				depthParens++
			case ")":
				if depthParens > 0 {
					depthParens--
				}
			case ",":
				if depthBraces == 0 && depthBrackets == 0 && depthParens == 0 {
					return sourceEdit{}, false
				}
			case ";":
				if depthBraces == 0 && depthBrackets == 0 && depthParens == 0 {
					statementEnd = i
				}
			}
		}
		if statementEnd >= 0 {
			break
		}
	}
	if statementEnd < 0 || strings.ContainsAny(source[tokens[declaration].start:tokens[statementEnd].start], "\r\n") || declarationHasConflictingBinding(tokens, bodyOpen, bodyClose, declaration, name) {
		return sourceEdit{}, false
	}
	for i := declaration + 2; i < statementEnd; i++ {
		if isBindingReference(tokens, i, name) {
			return sourceEdit{}, false
		}
	}

	for i := bodyOpen + 1; i < declaration; i++ {
		if isBindingReference(tokens, i, name) {
			return sourceEdit{}, false
		}
	}
	for i := declaration + 2; i < bodyClose; i++ {
		if !isBindingReference(tokens, i, name) {
			continue
		}
		if insideNestedFunction(tokens, declaration, bodyClose, i) {
			return sourceEdit{}, false
		}
	}

	kind := "const"
	for i := statementEnd + 1; i < bodyClose; i++ {
		if !isBindingReference(tokens, i, name) {
			continue
		}
		if isBindingWrite(tokens, i) {
			kind = "let"
			break
		}
	}
	return sourceEdit{start: tokens[declaration].start, end: tokens[declaration].end, replacement: kind}, true
}

func isUnsafeLexicalBindingName(name string) bool {
	switch name {
	case "arguments", "eval", "await", "yield", "enum", "implements", "interface", "let", "package", "private", "protected", "public", "static":
		return true
	default:
		return false
	}
}

func functionParameterNamed(tokens []jsToken, open, close int, name string) bool {
	for i := open + 1; i < close; i++ {
		if tokens[i].kind == identifierToken && tokens[i].value == name {
			return true
		}
	}
	return false
}

func declarationHasConflictingBinding(tokens []jsToken, bodyOpen, bodyClose, declaration int, name string) bool {
	for i := bodyOpen + 1; i+1 < bodyClose; i++ {
		if i == declaration || tokens[i+1].kind != identifierToken || tokens[i+1].value != name {
			continue
		}
		if is(tokens, i, "var") || is(tokens, i, "let") || is(tokens, i, "const") || is(tokens, i, "function") {
			return true
		}
	}
	return false
}

func isBindingReference(tokens []jsToken, index int, name string) bool {
	return index >= 0 && index < len(tokens) && tokens[index].kind == identifierToken && tokens[index].value == name
}

func insideNestedFunction(tokens []jsToken, declaration, bodyClose, reference int) bool {
	for i := declaration + 1; i < reference; i++ {
		if !is(tokens, i, "function") {
			continue
		}
		_, _, nestedOpen, nestedClose, ok, err := functionBounds(tokens, i)
		if err == nil && ok && nestedOpen < reference && reference < nestedClose && nestedClose < bodyClose {
			return true
		}
	}
	return false
}

func isBindingWrite(tokens []jsToken, index int) bool {
	if index > 0 && (tokens[index-1].value == "++" || tokens[index-1].value == "--") {
		return true
	}
	if index+1 >= len(tokens) {
		return false
	}
	switch tokens[index+1].value {
	case "=", "+=", "-=", "*=", "/=", "%=", "**=", "<<=", ">>=", ">>>=", "&=", "|=", "^=", "&&=", "||=", "??=", "++", "--", "in", "of":
		return true
	}
	if (tokens[index+1].value == "}" || tokens[index+1].value == "]") && index+2 < len(tokens) && tokens[index+2].value == "=" {
		return true
	}
	return false
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
