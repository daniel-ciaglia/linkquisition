package freedesktop

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"
)

type XdgService struct {
}

func (x *XdgService) GetDesktopEntryPathForFilename(name string) (string, error) {
	paths := x.GetApplicationPaths()

	if len(paths) == 0 {
		return "", fmt.Errorf("no valid desktop entry paths found in $XDG_DATA_DIRS")
	}

	for _, path := range paths {
		desktopEntryPath := filepath.Join(path, name)
		if _, err := os.Stat(desktopEntryPath); err == nil {
			return desktopEntryPath, nil
		}
	}

	return "", fmt.Errorf("no .desktop entry found for %s", name)
}

func (x *XdgService) GetDesktopEntryPathForBinary(binary string) (string, error) {
	paths := x.GetApplicationPaths()

	if len(paths) == 0 {
		return "", fmt.Errorf("no valid desktop entry paths found in $XDG_DATA_DIRS")
	}

	// as most entries will have this pattern: `/usr/bin/chromium --profile-directory=Default %U`
	// we have to remove the CLI args to find equivalent .desktop files
	binaryParts := shellSplit(binary)
	if len(binaryParts) == 0 {
		return "", fmt.Errorf("failed to parse binary string: empty input")
	}
	// The first part is the actual binary path
	binaryPath := binaryParts[0]

	// grep all the .desktop files in the paths for the binary basename and return the first match:
	pattern := fmt.Sprintf("^Exec=(%s|%s)", binaryPath, filepath.Base(binaryPath))
	grepArgs := []string{"-r", "-l", "-m", "1", "-E", pattern, "--include", "*.desktop"}
	grepArgs = append(grepArgs, paths...)
	cmd := exec.CommandContext(context.Background(), "grep", grepArgs...)

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		fmt.Println(out.String())
		return "", fmt.Errorf("failed to call grep for determining a .desktop entry for %s: %v", binary, err)
	}

	scanner := bufio.NewScanner(strings.NewReader(out.String()))

	if !scanner.Scan() {
		return "", fmt.Errorf("no .desktop entry found for %s", binary)
	}

	return scanner.Text(), nil
}

func (x *XdgService) GetApplicationPaths() []string {
	datadirs, isset := os.LookupEnv("XDG_DATA_DIRS")
	if !isset {
		datadirs = "/usr/local/share/:/usr/share/"
	}

	var paths []string

	for datadir := range strings.SplitSeq(datadirs, ":") {
		desktopEntryPath := filepath.Join(datadir, "applications")

		if _, err := os.Stat(desktopEntryPath); err == nil {
			paths = append(paths, desktopEntryPath)
		}
	}
	return paths
}

func (x *XdgService) SettingsCheck(property, subProperty string) (string, error) {
	cmd := exec.CommandContext(context.Background(), "xdg-settings", "check", property, subProperty)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to call xdg-settings check: %v", err)
	}

	// TODO not exatcly sure if this is The Way™
	return strings.Trim(out.String(), "\n"), nil
}

func (x *XdgService) SettingsGet(property string) (string, error) {
	cmd := exec.CommandContext(context.Background(), "xdg-settings", "get", property)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to call xdg-settings get: %v", err)
	}
	return out.String(), nil
}

func (x *XdgService) SettingsSet(property, value string) error {
	cmd := exec.CommandContext(context.Background(), "xdg-settings", "set", property, value)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to call xdg-settings set: %v", err)
	}
	return nil
}

// shellSplit splits a shell command string into tokens, respecting single and
// double quotes. This covers the common .desktop Exec field formats.
func shellSplit(s string) []string {
	var tokens []string
	var cur strings.Builder
	inSingle := false
	inDouble := false

	for i := 0; i < len(s); i++ {
		ch := rune(s[i])
		switch {
		case inSingle:
			if ch == '\'' {
				inSingle = false
			} else {
				cur.WriteRune(ch)
			}
		case inDouble:
			if ch == '\\' && i+1 < len(s) {
				i++
				cur.WriteByte(s[i])
			} else if ch == '"' {
				inDouble = false
			} else {
				cur.WriteRune(ch)
			}
		case ch == '\'':
			inSingle = true
		case ch == '"':
			inDouble = true
		case ch == '\\' && i+1 < len(s):
			i++
			cur.WriteByte(s[i])
		case unicode.IsSpace(ch):
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(ch)
		}
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}
