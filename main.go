package main

import (
	"embed"

	"github.com/sebjacobs/jotter/cmd"
)

//go:embed all:integrations
var skillsFS embed.FS

func main() {
	cmd.Execute(skillsFS)
}
