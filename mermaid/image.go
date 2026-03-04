package mermaid

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

// Protocol identifies the terminal image protocol to use.
type Protocol int

const (
	// ProtocolNone means the terminal does not support inline images.
	ProtocolNone Protocol = iota
	// ProtocolITerm2 uses the iTerm2 inline image protocol (OSC 1337).
	ProtocolITerm2
	// ProtocolKitty uses the Kitty graphics protocol.
	ProtocolKitty
)

// DetectProtocol checks environment variables to determine which inline image
// protocol the current terminal supports.
func DetectProtocol() Protocol {
	termProgram := os.Getenv("TERM_PROGRAM")
	switch strings.ToLower(termProgram) {
	case "iterm.app", "wezterm":
		return ProtocolITerm2
	case "ghostty":
		return ProtocolKitty
	}
	if os.Getenv("KITTY_PID") != "" {
		return ProtocolKitty
	}
	// When running inside tmux, TERM_PROGRAM is "tmux" but the actual
	// terminal can be detected via LC_TERMINAL or ITERM_SESSION_ID.
	if strings.ToLower(os.Getenv("LC_TERMINAL")) == "iterm2" || os.Getenv("ITERM_SESSION_ID") != "" {
		return ProtocolITerm2
	}
	return ProtocolNone
}

// ImageEscapeSequence reads a PNG file and returns the appropriate terminal
// escape sequence to display it inline.
func ImageEscapeSequence(pngPath string, widthCols int, protocol Protocol) (string, error) {
	data, err := os.ReadFile(pngPath)
	if err != nil {
		return "", fmt.Errorf("reading PNG: %w", err)
	}

	switch protocol {
	case ProtocolITerm2:
		return iterm2Sequence(data, widthCols), nil
	case ProtocolKitty:
		return kittySequence(data, widthCols), nil
	default:
		return "", fmt.Errorf("unsupported protocol: %d", protocol)
	}
}

func iterm2Sequence(data []byte, widthCols int) string {
	b64 := base64.StdEncoding.EncodeToString(data)
	// OSC 1337 ; File=[args] : base64data ST
	return fmt.Sprintf("\033]1337;File=inline=1;width=%d;preserveAspectRatio=1:%s\a", widthCols, b64)
}

func kittySequence(data []byte, widthCols int) string {
	b64 := base64.StdEncoding.EncodeToString(data)
	const chunkSize = 4096
	var sb strings.Builder
	for i := 0; i < len(b64); i += chunkSize {
		end := i + chunkSize
		if end > len(b64) {
			end = len(b64)
		}
		chunk := b64[i:end]
		more := 1
		if end >= len(b64) {
			more = 0
		}
		if i == 0 {
			// First chunk: include full header
			sb.WriteString(fmt.Sprintf("\033_Gf=100,a=T,c=%d,m=%d;%s\033\\", widthCols, more, chunk))
		} else {
			// Continuation chunk
			sb.WriteString(fmt.Sprintf("\033_Gm=%d;%s\033\\", more, chunk))
		}
	}
	return sb.String()
}

// FormatForViewport pads the image sequence with newlines for viewport spacing.
func FormatForViewport(imageSeq string, rows int) string {
	if rows <= 0 {
		return imageSeq
	}
	return imageSeq + strings.Repeat("\n", rows)
}
