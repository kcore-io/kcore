//go:build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const (
	binaryName = "kcore"

	GotestsumUrl       = "gotest.tools/gotestsum"
	GolangciLintUrl    = "github.com/golangci/golangci-lint/cmd/golangci-lint"
	HelmUrl            = "https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3"
	KindUrl            = "https://kind.sigs.k8s.io/dl/v0.22.0/kind-linux-amd64"
	KindConfigFile     = "config/dev/kind-config.yaml"
	KindClusterName    = "kafka"
	BitnamiHelmRepoUrl = "https://charts.bitnami.com/bitnami"

	HelmKafkaChartVersion   = "19.1.5"
	HelmKafkaChartNamespace = "kafka"
	HelmKafkaValuesFile     = "config/dev/kafka-values.yaml"
)

var (
	goexec = mg.GoCmd()
	g0     = sh.RunCmd(goexec)

	GreenMessage = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00c100"))
	// Border(lipgloss.NormalBorder(), true, false).
	// BorderForeground(lipgloss.Color("#00c100")).
	// Padding(0, 10, 0, 0)
)

func mustRun(cmd string, args ...string) {
	out := lipgloss.NewStyle().Bold(true).Render(
		fmt.Sprintf("\n> %s %s\n", cmd, strings.Join(args, " ")),
	)
	// if err != nil {
	// 	panic(err)
	// }

	fmt.Println(out)
	if err := sh.RunV(cmd, args...); err != nil {
		panic(err)
	}
}

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
		fmt.Printf("Installing gotestsum from %s\n", GotestsumUrl)
		mustRun(goexec, "install", GotestsumUrl)
	}
	if _, err := exec.LookPath("golangci-lint"); err != nil {
		fmt.Println("golangci-lint not found, installing...")
		fmt.Printf("Installing golangci-lint from %s\n", GolangciLintUrl)
		mustRun(goexec, "install", GolangciLintUrl)
	}
	if _, err := exec.LookPath("helm"); err != nil {
		fmt.Println("helm not found, installing...")
		fmt.Printf("Downloading helm installer from %s\n", HelmUrl)
		mustRun("curl", "-fsSL", "-o", "/tmp/get_helm.sh", HelmUrl)
		mustRun("chmod", "700", "/tmp/get_helm.sh")
		mustRun("/tmp/get_helm.sh")
	}

	if _, err := exec.LookPath("kind"); err != nil {
		if runtime.GOARCH == "amd64" && runtime.GOOS == "linux" {
			fmt.Println("kind not found, installing...")
			fmt.Printf("Downloading kind from %s\n", KindUrl)
			mustRun("curl", "-fsSLo", "./kind", KindUrl)
			mustRun("chmod", "+x", "./kind")
			fmt.Println("Running sudo mv ./kind /usr/local/bin/kind")
			mustRun("sudo", "mv", "./kind", "/usr/local/bin/kind")
		} else {
			fmt.Println("kind not found. We only support installing kind on linux amd64, please install it manually.")
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

func printFile(fName, fType string) error {

	fContent, err := sh.Output("cat", fName)
	if err != nil {
		return err
	}

	fRendered, err := glamour.Render(fmt.Sprintf("## %s\n\n```%s\n%s```", fName, fType, fContent), "dark")

	if err != nil {
		return err
	}
	fmt.Println(fRendered)

	return nil
}

func step(message string) {
	fmt.Println("")
	fmt.Println(
		lipgloss.NewStyle().Bold(true).
			Background(lipgloss.Color("#6a6b72")).
			Render(
				fmt.Sprintf("[Step] %s", message),
			),
	)
	// fmt.Println("")
}

func subStep(message string) {
	fmt.Println("")
	fmt.Println(
		lipgloss.NewStyle().Bold(true).
			Render(
				fmt.Sprintf("> %s", message),
			),
	)
}

// Infra deploys the infrastructure
func Infra() error {
	mg.Deps(checkTools)
	// instruction("Deploying infrastructure...")
	step(
		fmt.Sprintf(
			"Creating kind cluster \"%s\" using config file \"%s\"", KindClusterName, KindConfigFile,
		),
	)
	if err := printFile(KindConfigFile, "yaml"); err != nil {
		return err
	}

	mustRun("kind", "create", "cluster", "--name", KindClusterName, "--config", KindConfigFile)

	step(
		fmt.Sprintf(
			"Deploying kafka helm chart \"Version\": %s\t\"Namespace\": %s", HelmKafkaChartVersion,
			HelmKafkaChartNamespace,
		),
	)
	if s, err := sh.Output("helm", "repo", "list"); err == nil && strings.Contains(s, "bitnami") {
		subStep("Found bitnami helm repo, skipping adding it")
	} else if err == nil {
		subStep(fmt.Sprintf("Adding bitnami helm repo %s", BitnamiHelmRepoUrl))
		mustRun("helm", "repo", "add", "bitnami", BitnamiHelmRepoUrl)
	} else {
		return err
	}
	subStep(fmt.Sprintf("Installing Helm chart with values file \"%s\"", HelmKafkaValuesFile))
	if err := printFile(HelmKafkaValuesFile, "yaml"); err != nil {
		return err
	}
	mustRun(
		"helm", "install", "--create-namespace", "--version", HelmKafkaChartVersion, "--namespace",
		HelmKafkaChartNamespace, "--values", HelmKafkaValuesFile, "kafka", "bitnami/kafka",
	)
	instructions := `
---
# Infrastructure deployed successfully.

The Kafka broker(s) should be available at the domain and port specified in the kafka-values.yaml file
(e.g. kafka.local:30092).

> Note: The domain name must be added to /etc/hosts file to resolve to 127.0.0.1.`
	out, err := glamour.Render(fmt.Sprintf("%s", instructions), "dark")
	if err != nil {
		return err
	}
	fmt.Println(out)
	return nil
}
