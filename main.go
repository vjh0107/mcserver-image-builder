package main

import (
	"os"

	"go.junhyung.kr/mcserver-image-builder/internal/cli"
)

func main() {
	if err := cli.NewRootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}
