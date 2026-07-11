package factory

import (
	"errors"
	"time"

	"github.com/Justi/projectseapig/runners/gorunner"
	"github.com/Justi/projectseapig/runners/javarunner"
	"github.com/Justi/projectseapig/runners/jsrunner"
	"github.com/Justi/projectseapig/runners/pythonrunner"

	"github.com/Justi/projectseapig/runners"
)

func Pigtype(lang string) (runners.TestRunner, error) {
	switch lang {
	case "java":
		return &javarunner.Javatester{
			BinPath:  "mvn",
			BaseArgs: []string{"test"},
			Timeout:  5 * time.Second,
		}, nil
	case "js":
		return &jsrunner.JStester{
			BinPath:  "npm",
			BaseArgs: []string{"test", "--"},
			Timeout:  5 * time.Second,
		}, nil
	case "go":
		return &gorunner.Gotester{
			BinPath:  "go",
			BaseArgs: []string{"test"},
			Timeout:  5 * time.Second,
		}, nil
	case "python":
		// Using pytest as the default execution tool
		return &pythonrunner.Pythontester{
			BinPath:  "pytest",
			BaseArgs: []string{},
			Timeout:  5 * time.Second,
		}, nil
	default:
		return nil, errors.New("Lang not supported...")
	}
}
