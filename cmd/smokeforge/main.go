package main

import (
	"fmt"
	"os"

	"github.com/BoatengPeter/smokeforge/internal/cli"
)

func main() {
	if err := cli.NewRootCmd(os.Stdout, os.Stderr).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
