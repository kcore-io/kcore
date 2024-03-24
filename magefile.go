//go:build mage

package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const (
	binaryName = "kcore"
)

var goexec = "go"
var g0 = sh.RunCmd(goexec)

var Default = Check

// Build builds the binary
func Build() error {
	fmt.Println("Building...")
	return g0("build", "-o", binaryName, ".")
}

// Install installs the built binary
func Install() error {
	fmt.Println("Installing...")
	return g0("install", "-o", binaryName, ".")
}

// Clean cleans up the built binary
func Clean() {
	fmt.Println("Cleaning...")
	os.RemoveAll(binaryName)
}

// Run runs the code without building a binary first
func Run() {
	cmdArgs := append([]string{"run", "cmd/kcore/main.go"})
	if len(os.Args) > 2 {
		cmdArgs = append(cmdArgs, os.Args[2:]...)
	}
	// We want to always print the output of the command to stdout
	sh.RunV(goexec, cmdArgs...)
	return
}

func checkTools() error {
	if _, err := exec.LookPath("gotestsum"); err != nil {
		fmt.Println("gotestsum is not installed. Installing...")
		if err := g0("install", "gotest.tools/gotestsum"); err != nil {
			return err
		}
	}
	if _, err := exec.LookPath("golangci-lint"); err != nil {
		fmt.Println("golangci-lint not found, installing...")
		if err = g0("install", "github.com/golangci/golangci-lint/cmd/golangci-lint"); err != nil {
			return err
		}
	}
	return nil
}

// Test runs the tests
func Test() error {
	mg.Deps(checkTools)
	fmt.Println("Testing...")
	return sh.RunV("gotestsum", "-f", "standard-verbose", "./...")
}

// Lint runs the linter
func Lint() error {
	mg.Deps(checkTools)
	fmt.Println("Running golanci-lint linter...")
	return sh.RunV("golangci-lint", "run")
}

// Check a presubmit check that runs the tests and linter.
func Check() error {
	err := Lint()
	if err != nil {
		return err
	}
	return Test()
}
