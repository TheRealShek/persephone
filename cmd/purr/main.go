package main

import (
	"Persephone/cmd"
)

var Version = "dev"

func main() {
	cmd.SetVersion(Version)
	cmd.Execute()
}
