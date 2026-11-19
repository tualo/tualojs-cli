package compiler

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

const sourceMapBase64 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

type sourceMapV3 struct {
	Version        int      `json:"version"`
	File           string   `json:"file"`
	SourceRoot     string   `json:"sourceRoot"`
	Sources        []string `json:"sources"`
	SourcesContent []string `json:"sourcesContent"`
	Names          []string `json:"names"`
	Mappings       string   `json:"mappings"`
}

func assembleBundle(sources []sourceFile, normalizeLetConst, upgradeES5, includeSourceMap, minify bool) ([]byte, []byte, error) {
	var bundle strings.Builder
	mappings := sourceMapMappings{lines: []string{""}}
	generatedLine := 0
	write := func(text string) {
		bundle.WriteString(text)
		for i := 0; i < len(text); i++ {
			if text[i] == '\n' {
				generatedLine++
				mappings.ensureLine(generatedLine)
			}
		}
	}

	sourceMap := sourceMapV3{Version: 3, File: "app.js", SourceRoot: "/sources/", Names: []string{}}
	sourceIndexes := make(map[string]int)
	for _, source := range sources {
		content := string(source.data)
		if !source.raw && upgradeES5 {
			var err error
			content, err = upgradeES5JavaScript(content)
			if err != nil {
				return nil, nil, fmt.Errorf("upgrade ES5 source %s: %w", source.name, err)
			}
		}
		if !source.raw && normalizeLetConst {
			var err error
			content, err = transformJavaScript(content)
			if err != nil {
				return nil, nil, fmt.Errorf("transform source %s: %w", source.name, err)
			}
		}
		if minify && !source.raw {
			var err error
			content, err = minifyJavaScript(content)
			if err != nil {
				return nil, nil, fmt.Errorf("minify source %s: %w", source.name, err)
			}
		}
		write("\n// Source: " + source.name + "\n")

		if includeSourceMap {
			sourcePath := source.name
			index, exists := sourceIndexes[sourcePath]
			if !exists {
				index = len(sourceMap.Sources)
				sourceIndexes[sourcePath] = index
				sourceMap.Sources = append(sourceMap.Sources, sourcePath)
				original := source.originalData
				if original == nil {
					original = source.data
				}
				sourceMap.SourcesContent = append(sourceMap.SourcesContent, string(original))
			}
			originalTokens, originalErr := tokenize(string(source.data))
			generatedTokens, generatedErr := tokenize(content)
			if originalErr == nil && generatedErr == nil && tokenSequencesAlign(originalTokens, generatedTokens) {
				originalPositions := tokenPositions(string(source.data), originalTokens)
				generatedPositions := tokenPositions(content, generatedTokens)
				for i, generated := range generatedPositions {
					mappings.add(generatedLine+generated.line, generated.column, index, originalPositions[i].line, originalPositions[i].column)
				}
			} else {
				lines := strings.Split(content, "\n")
				if strings.HasSuffix(content, "\n") {
					lines = lines[:len(lines)-1]
				}
				if len(lines) == 0 {
					lines = []string{""}
				}
				for originalLine := range lines {
					mappings.add(generatedLine+originalLine, 0, index, originalLine, 0)
				}
			}
			write(content)
			if len(content) == 0 || content[len(content)-1] != '\n' {
				write("\n")
			}
		} else {
			write(content)
			if len(content) == 0 || content[len(content)-1] != '\n' {
				write("\n")
			}
		}
	}

	var sourceMapData []byte
	if includeSourceMap {
		write("//# sourceMappingURL=app.js.map\n")
		sourceMap.Mappings = strings.Join(mappings.lines, ";")
		encoded, err := json.Marshal(sourceMap)
		if err != nil {
			return nil, nil, fmt.Errorf("encode source map: %w", err)
		}
		sourceMapData = encoded
	}
	return []byte(bundle.String()), sourceMapData, nil
}

type mappedPosition struct {
	line   int
	column int
}

type sourceMapMappings struct {
	lines                      []string
	previousSource             int
	previousOriginalLine       int
	previousOriginalColumn     int
	previousGeneratedLine      int
	previousGeneratedColumn    int
	hasPreviousGeneratedColumn bool
}

func (m *sourceMapMappings) ensureLine(line int) {
	for len(m.lines) <= line {
		m.lines = append(m.lines, "")
	}
}

func (m *sourceMapMappings) add(generatedLine, generatedColumn, sourceIndex, originalLine, originalColumn int) {
	m.ensureLine(generatedLine)
	generatedDelta := generatedColumn
	if m.hasPreviousGeneratedColumn && m.previousGeneratedLine == generatedLine {
		generatedDelta -= m.previousGeneratedColumn
	}
	segment := encodeVLQ(generatedDelta) +
		encodeVLQ(sourceIndex-m.previousSource) +
		encodeVLQ(originalLine-m.previousOriginalLine) +
		encodeVLQ(originalColumn-m.previousOriginalColumn)
	if m.lines[generatedLine] != "" {
		m.lines[generatedLine] += ","
	}
	m.lines[generatedLine] += segment
	m.previousSource = sourceIndex
	m.previousOriginalLine = originalLine
	m.previousOriginalColumn = originalColumn
	m.previousGeneratedLine = generatedLine
	m.previousGeneratedColumn = generatedColumn
	m.hasPreviousGeneratedColumn = true
}

func tokenSequencesAlign(original, generated []jsToken) bool {
	if len(original) != len(generated) {
		return false
	}
	for i := range original {
		if original[i].kind == generated[i].kind && original[i].value == generated[i].value {
			continue
		}
		if original[i].kind == identifierToken && generated[i].kind == identifierToken &&
			(original[i].value == "let" || original[i].value == "const") && generated[i].value == "var" {
			continue
		}
		if original[i].kind == identifierToken && generated[i].kind == identifierToken &&
			original[i].value == "var" && (generated[i].value == "let" || generated[i].value == "const") {
			continue
		}
		return false
	}
	return true
}

func tokenPositions(source string, tokens []jsToken) []mappedPosition {
	positions := make([]mappedPosition, len(tokens))
	line, column, offset := 0, 0, 0
	for i, token := range tokens {
		for offset < token.start {
			r, size := utf8.DecodeRuneInString(source[offset:])
			if size == 0 {
				break
			}
			if r == '\n' || r == '\r' && (offset+size >= len(source) || source[offset+size] != '\n') {
				line++
				column = 0
			} else if r != '\r' {
				column += utf16.RuneLen(r)
			}
			offset += size
		}
		positions[i] = mappedPosition{line: line, column: column}
	}
	return positions
}

func encodeVLQ(value int) string {
	encoded := value << 1
	if value < 0 {
		encoded = ((-value) << 1) | 1
	}
	var result strings.Builder
	for {
		digit := encoded & 31
		encoded >>= 5
		if encoded > 0 {
			digit |= 32
		}
		result.WriteByte(sourceMapBase64[digit])
		if encoded == 0 {
			break
		}
	}
	return result.String()
}
