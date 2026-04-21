package main

import (
	"fmt"
	"os"

	"github.com/kluctl/go-embed-python/pip"
)

func main() {
	targetDir := "./python/bundled"

	fmt.Println("Bundling ansible-core and ansible.posix collection...")

	err := pip.CreateEmbeddedPipPackagesForKnownPlatforms("requirements.txt", targetDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Bundling complete. Files written to:", targetDir)
}
