package mermaid

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Available reports whether the mmdc (Mermaid CLI) binary is on the PATH.
func Available() bool {
	_, err := exec.LookPath("mmdc")
	return err == nil
}

func cacheDir() string {
	return filepath.Join(os.TempDir(), "glow-mermaid")
}

// RenderToPNG renders a mermaid block to a PNG file, returning the path.
// Results are cached by source hash and width.
func RenderToPNG(block Block, widthPixels int) (string, error) {
	dir := cacheDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("creating cache dir: %w", err)
	}

	outPath := filepath.Join(dir, fmt.Sprintf("%s_%d.png", block.SourceHash, widthPixels))

	// Return cached result if it exists
	if _, err := os.Stat(outPath); err == nil {
		return outPath, nil
	}

	// Write the mermaid source to a temp file
	inPath := filepath.Join(dir, fmt.Sprintf("%s.mmd", block.SourceHash))
	if err := os.WriteFile(inPath, []byte(block.Source), 0o600); err != nil {
		return "", fmt.Errorf("writing mermaid source: %w", err)
	}
	defer os.Remove(inPath)

	// Run mmdc with a timeout to avoid hanging on puppeteer/chromium issues.
	// Detach stdin so puppeteer/chromium doesn't inherit the terminal's stdin
	// which can cause hangs.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "mmdc",
		"-i", inPath,
		"-o", outPath,
		"-w", fmt.Sprintf("%d", widthPixels),
		"--backgroundColor", "transparent",
		"--quiet",
	)
	cmd.Stdin = nil
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("mmdc: %w: %s", err, string(out))
	}

	return outPath, nil
}

// ClearCache removes all cached mermaid renders.
func ClearCache() {
	os.RemoveAll(cacheDir())
}
