package color

import "github.com/fatih/color"

var (
	Red    = color.New(color.FgRed).SprintFunc()
	Green  = color.New(color.FgGreen).SprintFunc()
	Yellow = color.New(color.FgHiYellow).SprintFunc()
)
