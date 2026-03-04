package mermaid

import (
	"fmt"
	"strings"
	"testing"
)

func TestExtractBlocks_None(t *testing.T) {
	md := "# Hello\nSome text\n```go\nfmt.Println()\n```\n"
	blocks := ExtractBlocks(md)
	if len(blocks) != 0 {
		t.Fatalf("expected 0 blocks, got %d", len(blocks))
	}
}

func TestExtractBlocks_Single(t *testing.T) {
	md := "# Test\n```mermaid\ngraph TD\n    A-->B\n```\n"
	blocks := ExtractBlocks(md)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	if blocks[0].Index != 0 {
		t.Errorf("expected index 0, got %d", blocks[0].Index)
	}
	if !strings.Contains(blocks[0].Source, "graph TD") {
		t.Errorf("expected source to contain 'graph TD', got %q", blocks[0].Source)
	}
	if blocks[0].SourceHash == "" {
		t.Error("expected non-empty hash")
	}
}

func TestExtractBlocks_Multiple(t *testing.T) {
	md := "# Doc\n```mermaid\ngraph TD\n    A-->B\n```\nSome text\n```mermaid\nsequenceDiagram\n    A->>B: Hello\n```\n"
	blocks := ExtractBlocks(md)
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
	if blocks[0].Index != 0 || blocks[1].Index != 1 {
		t.Errorf("unexpected indices: %d, %d", blocks[0].Index, blocks[1].Index)
	}
	if !strings.Contains(blocks[1].Source, "sequenceDiagram") {
		t.Errorf("expected second block source to contain 'sequenceDiagram'")
	}
	if blocks[0].SourceHash == blocks[1].SourceHash {
		t.Error("expected different hashes for different blocks")
	}
}

func TestPlaceholderRoundtrip(t *testing.T) {
	md := "# Test\n```mermaid\ngraph TD\n    A-->B\n```\nEnd\n"
	blocks := ExtractBlocks(md)
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}

	withPlaceholders := PreparePlaceholders(md, blocks)
	if strings.Contains(withPlaceholders, "```mermaid") {
		t.Error("placeholders should have replaced mermaid fence")
	}
	if !strings.Contains(withPlaceholders, "GLOWMERMAIDPH0") {
		t.Error("expected placeholder in output")
	}

	replacement := "[RENDERED_IMAGE]"
	replacements := map[int]string{0: replacement}
	result := ReplacePlaceholders(withPlaceholders, replacements)
	if !strings.Contains(result, replacement) {
		t.Error("expected replacement in final output")
	}
	if strings.Contains(result, "GLOWMERMAIDPH") {
		t.Error("placeholder should have been replaced")
	}
}

func TestDetectProtocol(t *testing.T) {
	tests := []struct {
		name     string
		envKey   string
		envValue string
		expected Protocol
	}{
		{"iTerm2", "TERM_PROGRAM", "iTerm.app", ProtocolITerm2},
		{"WezTerm", "TERM_PROGRAM", "WezTerm", ProtocolITerm2},
		{"Ghostty", "TERM_PROGRAM", "ghostty", ProtocolKitty},
		{"Kitty via PID", "KITTY_PID", "12345", ProtocolKitty},
		{"Unknown", "TERM_PROGRAM", "xterm", ProtocolNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore env
			origTP := getEnv("TERM_PROGRAM")
			origKP := getEnv("KITTY_PID")
			defer func() {
				restoreEnv("TERM_PROGRAM", origTP)
				restoreEnv("KITTY_PID", origKP)
			}()

			t.Setenv("TERM_PROGRAM", "")
			t.Setenv("KITTY_PID", "")
			t.Setenv("LC_TERMINAL", "")
			t.Setenv("ITERM_SESSION_ID", "")
			t.Setenv(tt.envKey, tt.envValue)

			got := DetectProtocol()
			if got != tt.expected {
				t.Errorf("DetectProtocol() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestPlaceholderMultiple(t *testing.T) {
	md := "# Doc\n```mermaid\ngraph TD\n    A-->B\n```\nMiddle\n```mermaid\nsequenceDiagram\n    A->>B: Hi\n```\nEnd\n"
	blocks := ExtractBlocks(md)
	withPlaceholders := PreparePlaceholders(md, blocks)

	for i := range blocks {
		ph := fmt.Sprintf("GLOWMERMAIDPH%d", i)
		if !strings.Contains(withPlaceholders, ph) {
			t.Errorf("expected %s in output", ph)
		}
	}

	replacements := map[int]string{
		0: "[IMG_0]",
		1: "[IMG_1]",
	}
	result := ReplacePlaceholders(withPlaceholders, replacements)
	if !strings.Contains(result, "[IMG_0]") || !strings.Contains(result, "[IMG_1]") {
		t.Error("expected both replacements in output")
	}
}

func TestReplacePlaceholders_WithANSI(t *testing.T) {
	// Simulate glamour inserting ANSI codes within the placeholder at char boundaries
	ansiMangled := "GLOWMERMAIDPH\x1b[0m\x1b[38;5;252m0"
	replacements := map[int]string{0: "[IMAGE]"}
	result := ReplacePlaceholders(ansiMangled, replacements)
	if !strings.Contains(result, "[IMAGE]") {
		t.Errorf("expected ANSI-aware replacement to work, got %q", result)
	}
	if strings.Contains(result, "GLOWMERMAIDPH") {
		t.Errorf("placeholder should have been replaced, got %q", result)
	}
}

func getEnv(key string) *string {
	v, ok := lookupEnv(key)
	if !ok {
		return nil
	}
	return &v
}

func lookupEnv(key string) (string, bool) {
	return "", false // placeholder for tests using t.Setenv
}

func restoreEnv(key string, val *string) {
	// t.Setenv handles cleanup automatically
}
