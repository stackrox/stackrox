package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func readUserInput(prompt string) (string, error) {
	printToStderr(prompt)
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

func readUserInputFromFlag(f *pflag.Flag) (string, error) {
	var prompt string
	if f.Value.String() != "" {
		prompt = fmt.Sprintf("Enter %s (default: '%s'): ", f.Usage, f.Value)
	} else {
		prompt = fmt.Sprintf("Enter %s: ", f.Usage)
	}

	text, err := readUserInput(prompt)
	if err != nil {
		return "", err
	}
	if text == "" {
		return f.Value.String(), nil
	}
	return text, nil
}

func readUserString(f *pflag.Flag) string {
	s, err := readUserInputFromFlag(f)
	if err != nil {
		printToStderr("Error reading value from command line. Please try again.\n")
		return readUserString(f)
	}
	return s
}

func printToStderr(t string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, t, args...)
}

func processFlag(f *pflag.Flag) string {
	userInput := readUserString(f)
	if userInput == "" {
		return ""
	}
	return fmt.Sprintf("--%s=%s", f.Name, userInput)
}

func choseCommand(prompt string, c *cobra.Command) (args []string) {
	for true {
		cmdString, err := readUserInput(prompt)
		if err != nil {
			printToStderr("\nCould not read user input. Did you specify '-i' in the Docker run command?\n")
			os.Exit(1)
		}
		if cmdString == "" {
			return
		}
		for _, subCommand := range c.Commands() {
			if subCommand.Name() == cmdString {
				args = append(args, walkTree(subCommand)...)
				return
			}
		}
		printToStderr("'%s' is not a valid option. Please try again.\n", cmdString)
	}
	return
}

func walkTree(c *cobra.Command) (args []string) {
	args = []string{c.Name()}
	c.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if val := processFlag(f); val != "" {
			args = append(args, val)
		}
	})
	c.Flags().VisitAll(func(f *pflag.Flag) {
		if val := processFlag(f); val != "" {
			args = append(args, val)
		}
	})

	// group commands by their annotation categories
	categoriesToCommands := make(map[string][]string)
	for _, cmd := range c.Commands() {
		if category, ok := cmd.Annotations["category"]; ok {
			categoriesToCommands[category] = append(categoriesToCommands[category], cmd.Name())
		}
	}

	for k, v := range categoriesToCommands {
		cmdPrompt := fmt.Sprintf("%s (%s): ", k, strings.Join(v, ", "))
		args = append(args, choseCommand(cmdPrompt, c)...)
	}
	return
}
