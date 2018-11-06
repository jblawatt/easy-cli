package main

import (
	ecmd "github.com/jblawatt/easy-cli/cmd"
)

func main() {
	if err := ecmd.RootCmd.Execute(); err != nil {
		panic(err)
	}
}
