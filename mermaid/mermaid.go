// Package mermaid provides extraction, rendering, and inline display of
// Mermaid diagram blocks found in Markdown documents.
package mermaid

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
)

// ansiRe matches ANSI escape sequences that glamour may insert within text.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

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
	// No underscores — glamour inserts ANSI reset/color sequences at underscore
	// boundaries which breaks literal string matching in ReplacePlaceholders.
	return fmt.Sprintf("GLOWMERMAIDPH%d", index)
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
// It handles the case where glamour may have inserted ANSI escape sequences
// within the placeholder text.
func ReplacePlaceholders(glamourOutput string, replacements map[int]string) string {
	result := glamourOutput
	for idx, replacement := range replacements {
		ph := placeholderFor(idx)
		// First try exact match (fast path)
		if strings.Contains(result, ph) {
			result = strings.ReplaceAll(result, ph, replacement)
			continue
		}
		// Fallback: glamour may have inserted ANSI codes within the placeholder.
		// Build a regex that allows ANSI sequences between each character.
		var pattern strings.Builder
		for i, ch := range ph {
			if i > 0 {
				pattern.WriteString(`(?:\x1b\[[0-9;]*m)*`)
			}
			pattern.WriteString(regexp.QuoteMeta(string(ch)))
		}
		re := regexp.MustCompile(pattern.String())
		result = re.ReplaceAllString(result, replacement)
	}
	return result
}
