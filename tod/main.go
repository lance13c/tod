package main

import "github.com/ciciliostudio/tod/cmd"

var version = "dev"

func main() {
	cmd.SetVersion(version)
	cmd.Execute()
}