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

	point, _ := golang.Detect(".")
	if point > 15 {
		return "go"
	}

	point2, _ := java.Detect(".")
	if point2 > 15 {
		return "java"
	}

	point3, _ := js.Detect(".")
	if point3 > 15 {
		return "js"
	}
	point4, _ := python.Detect(".")
	if point4 > 15 {
		return "python"
	}
	return ""
}
