package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

var (
	styleOnce sync.Once
	enabled   bool

	greenStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	redStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	yellowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	cyanStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	boldStyle   = lipgloss.NewStyle().Bold(true)
)

func Enabled() bool {
	styleOnce.Do(func() {
		if os.Getenv("NO_COLOR") != "" {
			enabled = false
			return
		}

		enabled = term.IsTerminal(int(os.Stdout.Fd()))
	})

	return enabled
}

func render(style lipgloss.Style, text string) string {
	if !Enabled() {
		return text
	}

	return style.Render(text)
}

func Added(text string) string {
	return render(greenStyle, text)
}

func Removed(text string) string {
	return render(redStyle, text)
}

func Modified(text string) string {
	return render(yellowStyle, text)
}

func Info(text string) string {
	return render(cyanStyle, text)
}

func Metadata(text string) string {
	return render(dimStyle, text)
}

func SectionHeader(text string) string {
	return render(boldStyle, text)
}

func ErrorText(text string) string {
	return render(redStyle, text)
}

func ErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	return ErrorText(err.Error())
}

func Successf(format string, args ...any) string {
	return Added(fmt.Sprintf(format, args...))
}

func Infof(format string, args ...any) string {
	return Info(fmt.Sprintf(format, args...))
}

func Metadataf(format string, args ...any) string {
	return Metadata(fmt.Sprintf(format, args...))
}

func Errorf(format string, args ...any) string {
	return ErrorText(fmt.Sprintf(format, args...))
}

func DiffAddedLine(line string) string {
	return Added(line)
}

func DiffRemovedLine(line string) string {
	return Removed(line)
}

func DiffHunkHeader(line string) string {
	return Info(line)
}

func StatusPrefix(code string) string {
	switch code {
	case "A":
		return Added(code)
	case "D":
		return Removed(code)
	case "M":
		return Modified(code)
	case "?":
		return Info(code)
	default:
		return code
	}
}

func BranchName(name string) string {
	return Info(name)
}

func DirectoryName(name string) string {
	return Info(name)
}

func FileName(name string) string {
	return name
}

func StyledPath(path string) string {
	if path == "" || !Enabled() {
		return path
	}

	separator := string(filepath.Separator)
	parts := strings.Split(path, separator)
	if len(parts) == 1 {
		return FileName(parts[0])
	}

	for i := 0; i < len(parts)-1; i++ {
		if parts[i] != "" {
			parts[i] = DirectoryName(parts[i])
		}
	}
	parts[len(parts)-1] = FileName(parts[len(parts)-1])

	return strings.Join(parts, separator)
}
