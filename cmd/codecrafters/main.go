package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/codecrafters-io/cli/internal/commands"
	"github.com/codecrafters-io/cli/internal/utils"
	"github.com/fatih/color"
)

// Usage: codecrafters test
func main() {
	utils.InitSentry()
	defer utils.TeardownSentry()

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `CLI to interact with CodeCrafters

USAGE
  $ codecrafters [command]

EXAMPLES
  $ codecrafters test              # Run tests without committing changes

VERSION
  %s
`, utils.VersionString())

	}

	help := flag.Bool("help", false, "show usage instructions")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	if *showVersion {
		fmt.Println(utils.VersionString())
		os.Exit(0)
	}

	err := run()
	if err != nil {
		red := color.New(color.FgRed).SprintFunc()

		if err.Error() != "" {
			fmt.Fprintf(os.Stderr, "%v\n", red(err))
		}

		os.Exit(1)
	}

	os.Exit(0)
}

func run() error {
	ctx := context.Background()
	logger := utils.NewLogger()
	cmd := flag.Arg(0)

	logger.Debug().Msgf("Running command: %s", cmd)

	ctx = logger.WithContext(ctx)

	switch cmd {
	case "test":
		testCmd := flag.NewFlagSet("test", flag.ExitOnError)
		shouldTestPrevious := testCmd.Bool("previous", false, "run tests for the current stage and all previous stages in ascending order")
		testCmd.Parse(flag.Args()[1:]) // parse the args after the test command

		return commands.TestCommand(ctx, *shouldTestPrevious)
	case "help",
		"": // no argument
		flag.Usage()
	default:
		red := color.New(color.FgRed).SprintFunc()
		fmt.Printf(red("Unknown command '%s'. Did you mean to run `codecrafters test`?\n\n"), cmd)
		fmt.Printf("Run `codecrafters help` for a list of available commands.\n")

		return fmt.Errorf("")
	}

	return nil
}
