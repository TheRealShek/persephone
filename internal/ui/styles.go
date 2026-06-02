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

	// Vibrant, modern hex-based terminal color palette.
	//
	// Semantic Styling System:
	// Rather than using arbitrary terminal color index IDs (which vary widely across configurations),
	// we bind strict HSL-tailored premium Hex tones to standard VCS states:
	//  - Emerald Green: Successful operations & added/staged items
	//  - Coral Red: Hard errors, failures & deleted/removed items
	//  - Amber Orange: Changed metadata warning states & modified files
	//  - Electric Cyan: Information banners, helper tags & standard file permissions
	//  - Cool Slate Gray: Descriptive footnotes, SHA hash prefixes & minor timestamps
	greenStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")).Bold(true) // Emerald Green
	redStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Bold(true) // Coral Red
	yellowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Bold(true) // Amber Orange
	cyanStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#06B6D4")).Bold(true) // Electric Cyan
	blueStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#3B82F6")).Bold(true) // Ocean Blue Hint
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#64748B"))            // Cool Slate Gray
	boldStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F1F5F9"))
	headerStyle = lipgloss.NewStyle().Bold(true).Underline(true)
	branchStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#A855F7")).Bold(true)
	dirStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#64748B"))
	pathStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#3B82F6"))

	successBadge = lipgloss.NewStyle().Background(lipgloss.Color("#22C55E")).Foreground(lipgloss.Color("#000000")).Bold(true)
	infoBadge    = lipgloss.NewStyle().Background(lipgloss.Color("#06B6D4")).Foreground(lipgloss.Color("#000000")).Bold(true)
	hintBadge    = lipgloss.NewStyle().Background(lipgloss.Color("#3B82F6")).Foreground(lipgloss.Color("#000000")).Bold(true)
	errorBadge   = lipgloss.NewStyle().Background(lipgloss.Color("#EF4444")).Foreground(lipgloss.Color("#000000")).Bold(true)

	// Command Help and Ls formatting styles
	whiteStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
	plainWhiteStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	faintWhiteStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Faint(true)
	cmdHelpStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")).MarginLeft(2).Width(10)
	flagHelpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#22C55E")).MarginLeft(2).Width(20)
	lsHeaderStyle     = cyanStyle.Copy().Width(20)
	lsHashHeaderStyle = cyanStyle.Copy().Width(11)
	lsFileStyle       = plainWhiteStyle.Copy().Width(20)
	lsHashStyle       = dimStyle.Copy().Width(11)
)

// Enabled determines if the stdout session is capable of displaying ANSI Escape sequences.
//
// Accessibility & Pipelining Invariants:
// We explicitly check standard os.Getenv("NO_COLOR"). This is a critical VCS design contract:
// if a developer redirects purr output to a file (e.g. `purr ls > output.txt`), or runs a CI/CD job,
// the terminal capability check fails or NO_COLOR skips rendering, ensuring output files contain
// clean plain text rather than ANSI control sequences.
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

// render applies styling to target text, falling back to plain text if terminal capabilities are missing.
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

// Help and List UI Formatters

func HelpTagline(text string) string {
	return render(whiteStyle, text)
}

func HelpSection(text string) string {
	return render(cyanStyle, text)
}

func HelpGroup(text string) string {
	return render(yellowStyle, text)
}

func HelpCommand(name, short, usage string) string {
	paddedShort := fmt.Sprintf("%-29s", short)
	if !Enabled() {
		return fmt.Sprintf("  %-10s  %s(%s)", name, paddedShort, usage)
	}
	return cmdHelpStyle.Render(name) + "  " + faintWhiteStyle.Render(paddedShort) + faintWhiteStyle.Render("("+usage+")")
}

func HelpFlag(name, usage string) string {
	if !Enabled() {
		return fmt.Sprintf("  %-20s  %s", name, usage)
	}
	return flagHelpStyle.Render(name) + "  " + dimStyle.Render(usage)
}

func HelpFooter(text string) string {
	return render(dimStyle, text)
}

func LsHeader() string {
	if !Enabled() {
		return fmt.Sprintf("%-20s%-11s%s", "FILE", "HASH", "PERM")
	}
	return lsHeaderStyle.Render("FILE") + lsHashHeaderStyle.Render("HASH") + cyanStyle.Render("PERM")
}

func LsRow(file, hash, perm string) string {
	if !Enabled() {
		return fmt.Sprintf("%-20s%-11s%s", file, hash, perm)
	}
	return lsFileStyle.Render(file) + lsHashStyle.Render(hash) + dimStyle.Render(perm)
}

// LogCommitHeader highlights the content-addressed identifier that links each displayed history node
// back to its loose object under `.purr/objects`.
func LogCommitHeader(hash string) string {
	return Info("commit") + " " + Added(hash)
}

func LogLabel(label string) string {
	return Metadata(label)
}

func LogMessage(message string) string {
	lines := strings.Split(message, "\n")
	for i := range lines {
		lines[i] = "    " + lines[i]
	}
	return strings.Join(lines, "\n")
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
	msg := fmt.Sprintf(format, args...)
	if !Enabled() {
		return "[SUCCESS] " + msg
	}
	return greenStyle.Render("[SUCCESS]") + " " + msg
}

func Infof(format string, args ...any) string {
	msg := fmt.Sprintf(format, args...)
	if !Enabled() {
		return "[INFO]    " + msg
	}
	return cyanStyle.Render("[INFO]") + "    " + msg
}

func Hintf(format string, args ...any) string {
	msg := fmt.Sprintf(format, args...)
	if !Enabled() {
		return "[HINT]    " + msg
	}
	return blueStyle.Render("[HINT]") + "    " + msg
}

func Warningf(format string, args ...any) string {
	msg := fmt.Sprintf(format, args...)
	if !Enabled() {
		return "[WARNING] " + msg
	}
	return yellowStyle.Render("[WARNING]") + " " + msg
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
	msg := fmt.Sprintf(format, args...)
	if !Enabled() {
		return "[ERROR]   " + msg
	}
	return redStyle.Render("[ERROR]") + "   " + msg
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

// StyledPath decomposes a workspace path and highlights directory steps and filenames differently.
//
// Visual Parsing Engine:
// We parse the path using the platform-appropriate path separator. Intermediate directories
// are highlighted with directory styles (`dirStyle` muted sky blue), while the terminal node is highlighted
// with filename styles (`pathStyle` vibrant ocean blue). This provides premium legibility,
// helping developers scanning massive CLI outputs easily distinguish package contexts from actual files.
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
