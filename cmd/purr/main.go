package main

import (
	"persephone/cmd"
)

var Version = "dev"

func main() {
	cmd.SetVersion(Version)
	cmd.Execute()
}
