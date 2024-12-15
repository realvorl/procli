package pkg

import (
	"fmt"

	"github.com/fatih/color"
)

// PrintCheckResult prints the result with appropriate icon and color
func PrintCheckResult(item string, status bool) {
	var icon string
	var output string

	if status {
		icon = "✅"
		output = color.GreenString("%s %s", icon, item)
	} else {
		icon = "❌"
		output = color.RedString("%s %s", icon, item)
	}

	fmt.Println(output)
}
