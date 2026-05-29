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

	// Vibrant, modern hex-based terminal color palette
	greenStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")).Bold(true) // Emerald Green
	redStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Bold(true) // Coral Red
	yellowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Bold(true) // Amber Orange
	cyanStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#06B6D4")).Bold(true) // Electric Cyan
	blueStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#3B82F6")).Bold(true) // Ocean Blue Hint
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#64748B"))            // Cool Slate Gray
	boldStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F1F5F9"))

	// Custom UI Element styles
	branchStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#C084FC")).Bold(true) // Violet Branch
	pathStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#3B82F6")).Bold(true) // Ocean Blue Path
	dirStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#60A5FA"))            // Muted Sky Blue Directories
	successBadge = lipgloss.NewStyle().Background(lipgloss.Color("#15803D")).Foreground(lipgloss.Color("#FFFFFF")).Padding(0, 1).Bold(true)
	infoBadge    = lipgloss.NewStyle().Background(lipgloss.Color("#0369A1")).Foreground(lipgloss.Color("#FFFFFF")).Padding(0, 1).Bold(true)
	hintBadge    = lipgloss.NewStyle().Background(lipgloss.Color("#2563EB")).Foreground(lipgloss.Color("#FFFFFF")).Padding(0, 1).Bold(true)
	errorBadge   = lipgloss.NewStyle().Background(lipgloss.Color("#B91C1C")).Foreground(lipgloss.Color("#FFFFFF")).Padding(0, 1).Bold(true)
	headerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#F472B6")).Bold(true).Underline(true) // Rose Pink Accent
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
	return render(headerStyle, text)
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
	if !Enabled() {
		return fmt.Sprintf(format, args...)
	}
	badge := successBadge.Render(" SUCCESS ")
	text := greenStyle.Render(fmt.Sprintf(format, args...))
	return fmt.Sprintf("%s %s", badge, text)
}

func Infof(format string, args ...any) string {
	if !Enabled() {
		return fmt.Sprintf(format, args...)
	}
	badge := infoBadge.Render(" INFO ")
	text := cyanStyle.Render(fmt.Sprintf(format, args...))
	return fmt.Sprintf("%s %s", badge, text)
}

func Hintf(format string, args ...any) string {
	if !Enabled() {
		return fmt.Sprintf(format, args...)
	}
	badge := hintBadge.Render(" HINT ")
	text := blueStyle.Render(fmt.Sprintf(format, args...))
	return fmt.Sprintf("%s %s", badge, text)
}

type HintError struct {
	Err error
}

func (e *HintError) Error() string {
	return e.Err.Error()
}

func NewHintError(err error) error {
	return &HintError{Err: err}
}

func Metadataf(format string, args ...any) string {
	return Metadata(fmt.Sprintf(format, args...))
}

func Errorf(format string, args ...any) string {
	if !Enabled() {
		return fmt.Sprintf(format, args...)
	}
	badge := errorBadge.Render(" ERROR ")
	text := redStyle.Render(fmt.Sprintf(format, args...))
	return fmt.Sprintf("%s %s", badge, text)
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
	return render(branchStyle, name)
}

func DirectoryName(name string) string {
	return render(dirStyle, name)
}

func FileName(name string) string {
	return render(pathStyle, name)
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
