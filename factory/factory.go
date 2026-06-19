package factory

import (
	"errors"

	"github.com/Justi/projectseapig/runners/gorunner"
	"github.com/Justi/projectseapig/runners/javarunner"
	"github.com/Justi/projectseapig/runners/jsrunner"
	"github.com/Justi/projectseapig/runners/pythonrunner"

	"github.com/Justi/projectseapig/runners"
)

func Pigtype(lang string) (runners.TestRunner, error) {
	switch lang {
	case "java":
		return &javarunner.Javatester{}, nil

	case "js":
		return &jsrunner.JStester{}, nil
	case "go":
		return &gorunner.Gotester{}, nil
	case "python":
		return &pythonrunner.Pythontester{}, nil
	default:
		return nil, errors.New("Lang not supported...")
	}
}
