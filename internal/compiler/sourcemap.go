package compiler

import (
	"encoding/json"
	"fmt"
	"strings"
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

func assembleBundle(sources []sourceFile, normalizeLetConst, includeSourceMap bool) ([]byte, []byte, error) {
	var bundle strings.Builder
	mappingLines := []string{""}
	generatedLine := 0
	write := func(text string) {
		bundle.WriteString(text)
		for i := 0; i < len(text); i++ {
			if text[i] == '\n' {
				generatedLine++
				if generatedLine == len(mappingLines) {
					mappingLines = append(mappingLines, "")
				}
			}
		}
	}

	sourceMap := sourceMapV3{Version: 3, File: "app.js", SourceRoot: "/sources/", Names: []string{}}
	sourceIndexes := make(map[string]int)
	previousSource, previousLine, previousColumn := 0, 0, 0
	for _, source := range sources {
		content := string(source.data)
		if normalizeLetConst {
			var err error
			content, err = transformJavaScript(content)
			if err != nil {
				return nil, nil, fmt.Errorf("transform source %s: %w", source.name, err)
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
			lines := strings.Split(content, "\n")
			if strings.HasSuffix(content, "\n") {
				lines = lines[:len(lines)-1]
			}
			if len(lines) == 0 {
				lines = []string{""}
			}
			for originalLine, line := range lines {
				segment := encodeVLQ(0) + encodeVLQ(index-previousSource) + encodeVLQ(originalLine-previousLine) + encodeVLQ(-previousColumn)
				mappingLines[generatedLine] = segment
				previousSource = index
				previousLine = originalLine
				previousColumn = 0
				write(line + "\n")
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
		sourceMap.Mappings = strings.Join(mappingLines, ";")
		encoded, err := json.Marshal(sourceMap)
		if err != nil {
			return nil, nil, fmt.Errorf("encode source map: %w", err)
		}
		sourceMapData = encoded
	}
	return []byte(bundle.String()), sourceMapData, nil
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
