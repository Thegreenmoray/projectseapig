package factory

import (
	"github.com/Justi/projectseapig/runners/gorunner"
	"github.com/Justi/projectseapig/runners/javarunner"
	"github.com/Justi/projectseapig/runners/jsrunner"
	"github.com/Justi/projectseapig/runners/pythonrunner"
)

func Lang() string {
	golang := gorunner.Gotester{}
	java := javarunner.Javatester{}
	js := jsrunner.JStester{}
	python := pythonrunner.Pythontester{}

	if golang.Detect(".") {
		return "go"
	}
	if java.Detect(".") {
		return "java"
	}
	if js.Detect(".") {
		return "js"
	}
	if python.Detect(".") {
		return "python"
	}
	return ""
}
