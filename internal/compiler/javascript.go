package compiler

import (
	"fmt"
	"strings"
)

type tokenKind byte

const (
	identifierToken  tokenKind = 'i'
	stringToken      tokenKind = 's'
	templateToken    tokenKind = 't'
	regexToken       tokenKind = 'r'
	numberToken      tokenKind = 'n'
	punctuationToken tokenKind = 'p'
)

type jsToken struct {
	kind  tokenKind
	value string
	start int
	end   int
}

type classDeclaration struct {
	name         string
	override     string
	aliases      []string
	dependencies []string
	uses         []string
}

func parseExtMetadata(source string) ([]classDeclaration, []string, []string, error) {
	tokens, err := tokenize(source)
	if err != nil {
		return nil, nil, nil, err
	}

	var declarations []classDeclaration
	var applicationDependencies []string
	var sourceDependencies []string
	for i := 0; i+3 < len(tokens); i++ {
		if (tokens[i].kind == identifierToken || tokens[i].kind == stringToken) && tokens[i].value == "xtype" && is(tokens, i+1, ":") && i+2 < len(tokens) && isStaticString(tokens, i+2, len(tokens)) {
			sourceDependencies = append(sourceDependencies, widgetAlias(tokens[i+2].value))
		}
		if i+2 < len(tokens) && (tokens[i].kind == identifierToken || tokens[i].kind == stringToken) && is(tokens, i+1, ":") {
			switch tokens[i].value {
			case "controller", "viewModel", "ptype":
				if tokens[i+2].kind == stringToken && isStaticString(tokens, i+2, len(tokens)) {
					prefix := map[string]string{"controller": "controller", "viewModel": "viewmodel", "ptype": "plugin"}[tokens[i].value]
					sourceDependencies = append(sourceDependencies, configAlias(tokens[i+2].value, prefix))
				}
			case "plugins":
				for _, plugin := range stringArray(tokens, i+2, len(tokens)) {
					sourceDependencies = append(sourceDependencies, configAlias(plugin, "plugin"))
				}
			}
		}
		if !is(tokens, i, "Ext") || !is(tokens, i+1, ".") {
			continue
		}

		switch tokens[i+2].value {
		case "define":
			if !is(tokens, i+3, "(") || i+5 >= len(tokens) || tokens[i+4].kind != stringToken || !is(tokens, i+5, ",") {
				continue
			}
			configObject, ok, err := defineConfigObject(tokens, i+6)
			if err != nil {
				return nil, nil, nil, err
			}
			if !ok {
				continue
			}
			properties, err := objectProperties(tokens, configObject)
			if err != nil {
				return nil, nil, nil, err
			}
			declarations = append(declarations, classDeclaration{
				name:         tokens[i+4].value,
				override:     first(properties["override"]),
				aliases:      metadataAliases(properties),
				dependencies: metadataDependencies(properties),
				uses:         properties["uses"],
			})
		case "application":
			if i+4 >= len(tokens) || !is(tokens, i+3, "(") || !is(tokens, i+4, "{") {
				continue
			}
			properties, err := objectProperties(tokens, i+4)
			if err != nil {
				return nil, nil, nil, err
			}
			applicationDependencies = append(applicationDependencies, metadataDependencies(properties)...)
			applicationDependencies = append(applicationDependencies, properties["uses"]...)
		case "create":
			if i+4 < len(tokens) && is(tokens, i+3, "(") {
				if tokens[i+4].kind == stringToken {
					sourceDependencies = append(sourceDependencies, stringArray(tokens, i+4, len(tokens))...)
				} else if is(tokens, i+4, "{") {
					properties, err := objectProperties(tokens, i+4)
					if err != nil {
						return nil, nil, nil, err
					}
					sourceDependencies = append(sourceDependencies, properties["xclass"]...)
				}
			}
		case "require", "syncRequire":
			if i+4 < len(tokens) && is(tokens, i+3, "(") {
				sourceDependencies = append(sourceDependencies, stringArray(tokens, i+4, len(tokens))...)
			}
		case "widget":
			if i+4 < len(tokens) && is(tokens, i+3, "(") && tokens[i+4].kind == stringToken && isStaticString(tokens, i+4, len(tokens)) {
				sourceDependencies = append(sourceDependencies, widgetAlias(tokens[i+4].value))
			}
		}
	}
	return declarations, applicationDependencies, uniqueStrings(sourceDependencies), nil
}

func defineConfigObject(tokens []jsToken, start int) (int, bool, error) {
	if is(tokens, start, "{") {
		return start, true, nil
	}
	if !is(tokens, start, "function") {
		return 0, false, nil
	}
	paren := start + 1
	if paren < len(tokens) && tokens[paren].kind == identifierToken {
		paren++
	}
	for paren < len(tokens) && !is(tokens, paren, "(") {
		paren++
	}
	if paren >= len(tokens) {
		return 0, false, nil
	}
	argumentsEnd, err := matchingDelimiter(tokens, paren, "(", ")")
	if err != nil {
		return 0, false, err
	}
	body := argumentsEnd + 1
	if !is(tokens, body, "{") {
		return 0, false, nil
	}
	bodyEnd, err := matchingDelimiter(tokens, body, "{", "}")
	if err != nil {
		return 0, false, err
	}
	depth := 1
	for i := body + 1; i < bodyEnd; i++ {
		if depth == 1 && is(tokens, i, "return") && is(tokens, i+1, "{") {
			return i + 1, true, nil
		}
		if tokens[i].kind != punctuationToken {
			continue
		}
		if tokens[i].value == "{" {
			depth++
		} else if tokens[i].value == "}" {
			depth--
		}
	}
	return 0, false, nil
}

func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func metadataDependencies(properties map[string][]string) []string {
	var dependencies []string
	for _, key := range []string{"extend", "requires", "mainView", "mixins"} {
		dependencies = append(dependencies, properties[key]...)
	}
	return dependencies
}

func metadataAliases(properties map[string][]string) []string {
	aliases := append([]string{}, properties["alias"]...)
	for _, xtype := range properties["xtype"] {
		aliases = append(aliases, widgetAlias(xtype))
	}
	return uniqueStrings(aliases)
}

func widgetAlias(name string) string {
	if strings.HasPrefix(name, "widget.") {
		return name
	}
	return "widget." + name
}

func configAlias(name, kind string) string {
	if strings.Contains(name, ".") {
		return name
	}
	return kind + "." + name
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		unique = append(unique, value)
	}
	return unique
}

func objectProperties(tokens []jsToken, open int) (map[string][]string, error) {
	close, err := matchingDelimiter(tokens, open, "{", "}")
	if err != nil {
		return nil, err
	}
	properties := make(map[string][]string)
	for i := open + 1; i < close; {
		if is(tokens, i, ",") {
			i++
			continue
		}
		if i+1 >= close || (tokens[i].kind != identifierToken && tokens[i].kind != stringToken) || !is(tokens, i+1, ":") {
			i = skipPropertyValue(tokens, i, close)
			continue
		}

		key := tokens[i].value
		value := i + 2
		switch key {
		case "extend", "mainView", "override", "xclass":
			if value < close && tokens[value].kind == stringToken && isStaticString(tokens, value, close) {
				properties[key] = append(properties[key], tokens[value].value)
			}
		case "requires", "alias", "xtype", "uses":
			properties[key] = append(properties[key], stringArray(tokens, value, close)...)
		case "mixins":
			properties[key] = append(properties[key], classReferenceValues(tokens, value, close)...)
		}
		i = skipPropertyValue(tokens, value, close)
	}
	return properties, nil
}

func classReferenceValues(tokens []jsToken, start, limit int) []string {
	if start >= limit {
		return nil
	}
	if is(tokens, start, "[") || tokens[start].kind == stringToken {
		return stringArray(tokens, start, limit)
	}
	if !is(tokens, start, "{") {
		return nil
	}
	end, err := matchingDelimiter(tokens, start, "{", "}")
	if err != nil || end > limit {
		return nil
	}
	var values []string
	for i := start + 1; i < end; {
		if is(tokens, i, ",") {
			i++
			continue
		}
		if i+1 >= end || (tokens[i].kind != identifierToken && tokens[i].kind != stringToken) || !is(tokens, i+1, ":") {
			i = skipPropertyValue(tokens, i, end)
			continue
		}
		value := i + 2
		if value < end && tokens[value].kind == stringToken && isStaticString(tokens, value, end) {
			values = append(values, tokens[value].value)
		} else {
			values = append(values, stringArray(tokens, value, end)...)
		}
		i = skipPropertyValue(tokens, value, end)
	}
	return uniqueStrings(values)
}

func stringArray(tokens []jsToken, start, limit int) []string {
	if start >= limit {
		return nil
	}
	if tokens[start].kind == stringToken {
		if isStaticString(tokens, start, limit) {
			return []string{tokens[start].value}
		}
		return nil
	}
	if !is(tokens, start, "[") {
		return nil
	}
	end, err := matchingDelimiter(tokens, start, "[", "]")
	if err != nil || end > limit {
		return nil
	}
	var values []string
	braces, brackets, parentheses := 0, 0, 0
	for i := start + 1; i < end; i++ {
		if tokens[i].kind == stringToken && braces == 0 && brackets == 0 && parentheses == 0 && (i+1 == end || is(tokens, i+1, ",")) {
			values = append(values, tokens[i].value)
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
	return values
}

func isStaticString(tokens []jsToken, index, limit int) bool {
	if index >= limit || tokens[index].kind != stringToken {
		return false
	}
	if index+1 >= limit {
		return true
	}
	return is(tokens, index+1, ",") || is(tokens, index+1, ")") || is(tokens, index+1, "}") || is(tokens, index+1, "]")
}

func skipPropertyValue(tokens []jsToken, start, limit int) int {
	braces, brackets, parentheses := 0, 0, 0
	for i := start; i < limit; i++ {
		if tokens[i].kind != punctuationToken {
			continue
		}
		switch tokens[i].value {
		case "{":
			braces++
		case "}":
			if braces == 0 {
				return i
			}
			braces--
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
		case ",":
			if braces == 0 && brackets == 0 && parentheses == 0 {
				return i + 1
			}
		}
	}
	return limit
}

func matchingDelimiter(tokens []jsToken, open int, opening, closing string) (int, error) {
	depth := 0
	for i := open; i < len(tokens); i++ {
		if tokens[i].kind != punctuationToken {
			continue
		}
		if tokens[i].value == opening {
			depth++
		} else if tokens[i].value == closing {
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}
	return 0, fmt.Errorf("unterminated JavaScript %q block", opening)
}

func is(tokens []jsToken, index int, value string) bool {
	if index < 0 || index >= len(tokens) || tokens[index].value != value {
		return false
	}
	if isPunctuation(value) {
		return tokens[index].kind == punctuationToken
	}
	return tokens[index].kind == identifierToken
}

func isPunctuation(value string) bool {
	switch value {
	case ".", "(", ")", "{", "}", "[", "]", ":", ",", ";", "!", "?", "&", "|", "+", "-", "*", "%", "~", "^", "<", ">", "=":
		return true
	default:
		return false
	}
}

func tokenize(source string) ([]jsToken, error) {
	var tokens []jsToken
	for i := 0; i < len(source); {
		c := source[i]
		if isSpace(c) {
			i++
			continue
		}
		if c == '/' && i+1 < len(source) && source[i+1] == '/' {
			i += 2
			for i < len(source) && source[i] != '\n' {
				i++
			}
			continue
		}
		if c == '/' && i+1 < len(source) && source[i+1] == '*' {
			end := strings.Index(source[i+2:], "*/")
			if end < 0 {
				return nil, fmt.Errorf("unterminated JavaScript block comment")
			}
			i += end + 4
			continue
		}
		if c == '/' && canStartRegex(tokens) {
			if end, ok := regexLiteralEnd(source, i); ok {
				tokens = append(tokens, jsToken{kind: regexToken, value: source[i:end], start: i, end: end})
				i = end
				continue
			}
		}
		if c == '\'' || c == '"' || c == '`' {
			start := i
			quote := c
			i++
			var value strings.Builder
			closed := false
			for i < len(source) {
				current := source[i]
				i++
				if current == '\\' {
					if i == len(source) {
						break
					}
					value.WriteByte(source[i])
					i++
					continue
				}
				if current == quote {
					closed = true
					break
				}
				value.WriteByte(current)
			}
			if !closed {
				return nil, fmt.Errorf("unterminated JavaScript string")
			}
			kind := stringToken
			if quote == '`' {
				kind = templateToken
			}
			tokens = append(tokens, jsToken{kind: kind, value: value.String(), start: start, end: i})
			continue
		}
		if isIdentifierStart(c) {
			start := i
			i++
			for i < len(source) && isIdentifierPart(source[i]) {
				i++
			}
			tokens = append(tokens, jsToken{kind: identifierToken, value: source[start:i], start: start, end: i})
			continue
		}
		if isNumberStart(source, i) {
			start := i
			i = consumeNumber(source, i)
			tokens = append(tokens, jsToken{kind: numberToken, value: source[start:i], start: start, end: i})
			continue
		}
		if end := punctuatorEnd(source, i); end > i+1 {
			tokens = append(tokens, jsToken{kind: punctuationToken, value: source[i:end], start: i, end: end})
			i = end
			continue
		}
		tokens = append(tokens, jsToken{kind: punctuationToken, value: string(c), start: i, end: i + 1})
		i++
	}
	return tokens, nil
}

func canStartRegex(tokens []jsToken) bool {
	if len(tokens) == 0 {
		return true
	}
	previous := tokens[len(tokens)-1]
	if previous.kind == identifierToken {
		switch previous.value {
		case "return", "case", "throw", "typeof", "instanceof", "in", "of", "yield", "await", "delete", "void", "new":
			return true
		default:
			return false
		}
	}
	switch previous.value {
	case "(", "[", "{", "=", "=>", ":", ",", ";", "!", "?", "&", "|", "+", "-", "*", "%", "~", "^", "<", ">":
		return true
	default:
		return false
	}
}

func isNumberStart(source string, index int) bool {
	return index < len(source) && (source[index] >= '0' && source[index] <= '9' || source[index] == '.' && index+1 < len(source) && source[index+1] >= '0' && source[index+1] <= '9')
}

func consumeNumber(source string, index int) int {
	if index+1 < len(source) && source[index] == '0' {
		switch source[index+1] {
		case 'x', 'X', 'b', 'B', 'o', 'O':
			index += 2
			for index < len(source) && (isIdentifierPart(source[index]) || source[index] == '_') {
				index++
			}
			return index
		}
	}
	for index < len(source) && (source[index] >= '0' && source[index] <= '9' || source[index] == '_') {
		index++
	}
	if index < len(source) && source[index] == '.' {
		index++
		for index < len(source) && (source[index] >= '0' && source[index] <= '9' || source[index] == '_') {
			index++
		}
	}
	if index < len(source) && (source[index] == 'e' || source[index] == 'E') {
		index++
		if index < len(source) && (source[index] == '+' || source[index] == '-') {
			index++
		}
		for index < len(source) && (source[index] >= '0' && source[index] <= '9' || source[index] == '_') {
			index++
		}
	}
	if index < len(source) && source[index] == 'n' {
		index++
	}
	return index
}

var jsPunctuators = []string{
	">>>=", "===", "!==", "**=", "&&=", "||=", "??=", "<<=", ">>=", "...",
	"=>", "==", "!=", "<=", ">=", "++", "--", "&&", "||", "??", "?.", "+=", "-=", "*=", "/=", "%=", "<<", ">>>", ">>", "**", "&=", "|=", "^=",
}

func punctuatorEnd(source string, start int) int {
	for _, punctuator := range jsPunctuators {
		if strings.HasPrefix(source[start:], punctuator) {
			return start + len(punctuator)
		}
	}
	return start + 1
}

func regexLiteralEnd(source string, start int) (int, bool) {
	inCharacterClass := false
	escaped := false
	for i := start + 1; i < len(source); i++ {
		current := source[i]
		if current == '\n' || current == '\r' {
			return 0, false
		}
		if escaped {
			escaped = false
			continue
		}
		if current == '\\' {
			escaped = true
			continue
		}
		if current == '[' {
			inCharacterClass = true
			continue
		}
		if current == ']' {
			inCharacterClass = false
			continue
		}
		if current == '/' && !inCharacterClass {
			i++
			for i < len(source) && isIdentifierPart(source[i]) {
				i++
			}
			return i, true
		}
	}
	return 0, false
}

func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\r' || c == '\n' }
func isIdentifierStart(c byte) bool {
	return c == '_' || c == '$' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}
func isIdentifierPart(c byte) bool { return isIdentifierStart(c) || c >= '0' && c <= '9' }
