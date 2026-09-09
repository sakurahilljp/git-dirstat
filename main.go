package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/sakurahilljp/git-dirstat/cmd"
	"github.com/sakurahilljp/git-dirstat/pkg/model"
)

func main() {
	rootCmd := cmd.NewRootCommand()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)

		var exitErr *model.ExitCodeError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}

		// Cobra / pflag syntax errors are User Input errors (Exit Code 2)
		errMsg := err.Error()
		if strings.Contains(errMsg, "unknown flag") ||
			strings.Contains(errMsg, "unknown shorthand flag") ||
			strings.Contains(errMsg, "flag needs an argument") ||
			strings.Contains(errMsg, "invalid argument") {
			os.Exit(2)
		}

		os.Exit(1)
	}
}
