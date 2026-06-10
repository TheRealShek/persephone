package objects

import (
	"Persephone/internal/config"
	"time"
)

type TreeEntries struct {
	Name     string
	Filename string
	Sha1Hex  string
	IsTree   bool
	Mode     string
}

type CommitObj struct {
	TreeHash   string
	ParentHash string
	Author     config.PurrConfig
	Committer  config.PurrConfig
	Message    string
	Timestamp  time.Time
}
