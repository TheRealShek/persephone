package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
)

// PickerKind represents the category of the item selected in the interactive list.
type PickerKind int

const (
	PickerFile PickerKind = iota
	PickerBranch
)

// PickerItem implements Charm's list.Item interface.
//
// Component Design Pattern:
// This struct provides a unified metadata packaging strategy that allows lists (Bubble Tea elements)
// to dynamically display branches, staged files, or tags using semantic colors without duplicating
// the layout logic of the rendering delegate.
type PickerItem struct {
	Kind   PickerKind
	Value  string
	Detail string
}

// Title extracts the display title, dynamically applying VCS terminal styling.
func (item PickerItem) Title() string {
	switch item.Kind {
	case PickerBranch:
		return BranchName(item.Value)
	case PickerFile:
		return StyledPath(item.Value)
	default:
		return item.Value
	}
}

// Description extracts supplementary information (e.g. last commit summary, file size).
func (item PickerItem) Description() string {
	return Metadata(item.Detail)
}

// FilterValue determines the target string matched during terminal list fuzzy searching.
func (item PickerItem) FilterValue() string {
	return strings.ToLower(item.Value + " " + item.Detail)
}

// NewProgressBar initializes a terminal-wide progress bar with default color gradients.
func NewProgressBar() progress.Model {
	return progress.New(progress.WithDefaultGradient())
}

// NewSpinner initializes a standard loader spinner for running concurrent VCS background jobs.
func NewSpinner() spinner.Model {
	return spinner.New()
}

// NewPicker constructs an interactive, filterable terminal select list with title styling.
func NewPicker(title string, items []list.Item) list.Model {
	model := list.New(items, list.NewDefaultDelegate(), 0, 0)
	model.Title = SectionHeader(title)
	return model
}

// NewBranchPicker wraps standard list construction specifically tailored for branch selection flows.
func NewBranchPicker(title string, branches []string) list.Model {
	items := make([]list.Item, 0, len(branches))
	for _, branch := range branches {
		items = append(items, PickerItem{Kind: PickerBranch, Value: branch})
	}

	return NewPicker(title, items)
}

// NewFilePicker wraps standard list construction specifically tailored for path selection lists.
func NewFilePicker(title string, files []string) list.Model {
	items := make([]list.Item, 0, len(files))
	for _, file := range files {
		items = append(items, PickerItem{Kind: PickerFile, Value: file})
	}

	return NewPicker(title, items)
}
