/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"github.com/Justi/projectseapig/cmd"
	"github.com/Justi/projectseapig/factory"
)

func main() {
	factory.InitLogger(false)
	cmd.Execute()
}
