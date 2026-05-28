package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
)

type PickerKind int

const (
	PickerFile PickerKind = iota
	PickerBranch
)

type PickerItem struct {
	Kind   PickerKind
	Value  string
	Detail string
}

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

func (item PickerItem) Description() string {
	return Metadata(item.Detail)
}

func (item PickerItem) FilterValue() string {
	return strings.ToLower(item.Value + " " + item.Detail)
}

func NewProgressBar() progress.Model {
	return progress.New(progress.WithDefaultGradient())
}

func NewSpinner() spinner.Model {
	return spinner.New()
}

func NewPicker(title string, items []list.Item) list.Model {
	model := list.New(items, list.NewDefaultDelegate(), 0, 0)
	model.Title = SectionHeader(title)
	return model
}

func NewBranchPicker(title string, branches []string) list.Model {
	items := make([]list.Item, 0, len(branches))
	for _, branch := range branches {
		items = append(items, PickerItem{Kind: PickerBranch, Value: branch})
	}

	return NewPicker(title, items)
}

func NewFilePicker(title string, files []string) list.Model {
	items := make([]list.Item, 0, len(files))
	for _, file := range files {
		items = append(items, PickerItem{Kind: PickerFile, Value: file})
	}

	return NewPicker(title, items)
}
