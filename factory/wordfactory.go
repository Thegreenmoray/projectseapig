package factory

import (
	"github.com/Justi/projectseapig/runners"
	"github.com/Justi/projectseapig/runners/gorunner"
	"github.com/Justi/projectseapig/runners/javarunner"
	"github.com/Justi/projectseapig/runners/jsrunner"
	"github.com/Justi/projectseapig/runners/pythonrunner"
)

func Lang(projectPath string) string {
	runners := map[string]runners.TestRunner{
		"go":     &gorunner.Gotester{},
		"java":   &javarunner.Javatester{},
		"js":     &jsrunner.JStester{},
		"python": &pythonrunner.Pythontester{},
	}

	bestLang := ""
	bestScore := 0

	for lang, runner := range runners {
		score, _ := runner.Detect(projectPath)
		if score > bestScore {
			bestScore = score
			bestLang = lang
		}
	}

	// Optional: require a minimum score
	if bestScore < 3 {
		return ""
	}

	return bestLang
}
