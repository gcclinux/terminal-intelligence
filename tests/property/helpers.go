package property

import (
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
)

// genValidFilename generates valid filenames for testing
// Includes various file types supported by the editor
func genValidFilename() gopter.Gen {
	return gen.OneGenOf(
		// Simple filenames
		gen.Identifier().Map(func(s string) string { return s + ".txt" }),
		gen.Identifier().Map(func(s string) string { return s + ".md" }),
		gen.Identifier().Map(func(s string) string { return s + ".sh" }),
		gen.Identifier().Map(func(s string) string { return s + ".bash" }),
		gen.Identifier().Map(func(s string) string { return s + ".ps1" }),
		
		// Filenames with subdirectories
		gen.Identifier().Map(func(s string) string { return "subdir/" + s + ".txt" }),
		gen.Identifier().Map(func(s string) string { return "nested/path/" + s + ".md" }),
	)
}
