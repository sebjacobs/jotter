package internal

import "github.com/fatih/color"

// Color printers for consistent terminal output.
var (
	Bold = color.New(color.Bold).SprintFunc()
	Dim  = color.New(color.Faint).SprintFunc()
)

// typeColors maps entry types to their display color.
var typeColors = map[string]*color.Color{
	"start":      color.New(color.FgGreen),
	"finish":     color.New(color.FgBlue),
	"break":      color.New(color.FgYellow),
	"checkpoint": color.New(color.FgCyan),
}

// ColorType returns the entry type string in its assigned color.
func ColorType(t string) string {
	if c, ok := typeColors[t]; ok {
		return c.Sprint(t)
	}
	return t
}
