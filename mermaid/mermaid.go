// Package mermaid provides extraction, rendering, and inline display of
// Mermaid diagram blocks found in Markdown documents.
package mermaid

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
)

// Block represents a single mermaid code block extracted from markdown.
type Block struct {
	Index      int
	Source     string
	SourceHash string
}

var mermaidFenceRe = regexp.MustCompile("(?ms)^```mermaid\\s*\n(.*?)^```\\s*$")

// ExtractBlocks finds all ```mermaid fenced code blocks in the markdown source.
func ExtractBlocks(markdown string) []Block {
	matches := mermaidFenceRe.FindAllStringSubmatchIndex(markdown, -1)
	blocks := make([]Block, 0, len(matches))
	for i, loc := range matches {
		// loc[2] and loc[3] are the submatch (diagram source) start/end
		source := markdown[loc[2]:loc[3]]
		hash := fmt.Sprintf("%x", sha256.Sum256([]byte(source)))
		blocks = append(blocks, Block{
			Index:      i,
			Source:     source,
			SourceHash: hash,
		})
	}
	return blocks
}

func placeholderFor(index int) string {
	return fmt.Sprintf("GLOW_MERMAID_PLACEHOLDER_%d", index)
}

// PreparePlaceholders replaces mermaid fenced blocks in the markdown with
// unique text placeholders so that glamour does not render them as code blocks.
func PreparePlaceholders(markdown string, blocks []Block) string {
	result := markdown
	for i := len(blocks) - 1; i >= 0; i-- {
		loc := mermaidFenceRe.FindAllStringSubmatchIndex(result, -1)
		if i >= len(loc) {
			continue
		}
		// Replace the entire fenced block (loc[i][0] to loc[i][1]) with placeholder
		result = result[:loc[i][0]] + placeholderFor(blocks[i].Index) + "\n" + result[loc[i][1]:]
	}
	return result
}

// ReplacePlaceholders swaps placeholder strings in glamour's rendered output
// with the provided replacement strings (e.g. terminal image escape sequences).
func ReplacePlaceholders(glamourOutput string, replacements map[int]string) string {
	result := glamourOutput
	for idx, replacement := range replacements {
		result = strings.ReplaceAll(result, placeholderFor(idx), replacement)
	}
	return result
}
