package main

import (
	"os"

	"github.com/yetmike/ytcap/cmd"
)

var Version = "dev"

func main() {
	cmd.SetVersion(Version)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
