package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	categoryAnnotation = "category"
	groupAnnotation    = "group"
)

var (
	flagGroupMap = map[string]*flagGroup{
		"central": {
			name:       "central",
			optional:   false,
			groupOrder: 0,
		},
		"monitoring": {
			name:        "monitoring",
			optional:    true,
			groupOrder:  1,
			groupPrompt: "Would you like to run the monitoring stack?",
		},
		"clairify": {
			name:        "clairify",
			optional:    true,
			groupOrder:  2,
			groupPrompt: "Would you like to run Clairify?",
		},
	}
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

func promptUserForSection(prompt string) (bool, error) {
	prompt += " [y/N]"
	text, err := readUserInput(prompt)
	if err != nil {
		return false, err
	}
	if text == "" {
		return false, nil
	}
	return strings.ToLower(text) == "y", nil
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

type flagGroup struct {
	name        string
	optional    bool
	groupOrder  int
	groupPrompt string
	flags       []*pflag.Flag
}

func getFirstFromStringSliceOrEmpty(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}

func flagGroups(flags []*pflag.Flag) []*flagGroup {
	groups := make(map[string]*flagGroup)
	for _, f := range flags {
		name := getFirstFromStringSliceOrEmpty(f.Annotations[groupAnnotation])
		group, ok := flagGroupMap[name]
		if !ok {
			group = &flagGroup{}
		}
		group.flags = append(group.flags, f)
		groups[name] = group
	}
	var groupList []*flagGroup
	for _, g := range groups {
		groupList = append(groupList, g)
	}
	sort.SliceStable(groupList, func(i, j int) bool {
		return groupList[i].groupOrder < groupList[j].groupOrder
	})
	return groupList
}

func walkTree(c *cobra.Command) (args []string) {
	args = []string{c.Name()}
	var allFlags []*pflag.Flag
	c.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		allFlags = append(allFlags, f)
	})
	c.Flags().VisitAll(func(f *pflag.Flag) {
		allFlags = append(allFlags, f)
	})

	// Sort and group flags by their annotations. Take into account if flag section is optional. If so, then prompt for if they want that section
	for _, fg := range flagGroups(allFlags) {
		if fg.optional {
			// prompt if they want that section
			wanted, err := promptUserForSection(fg.groupPrompt)
			if err != nil {
				logger.Fatalf("Error prompting for section: %v", err)
			}
			if !wanted {
				continue
			}
		}
		for _, flag := range fg.flags {
			if val := processFlag(flag); val != "" {
				args = append(args, val)
			}
		}
	}

	// group commands by their annotation categories
	categoriesToCommands := make(map[string][]string)
	for _, cmd := range c.Commands() {
		if category, ok := cmd.Annotations[categoryAnnotation]; ok {
			categoriesToCommands[category] = append(categoriesToCommands[category], cmd.Name())
		}
	}

	for k, v := range categoriesToCommands {
		cmdPrompt := fmt.Sprintf("%s (%s): ", k, strings.Join(v, ", "))
		args = append(args, choseCommand(cmdPrompt, c)...)
	}
	return
}
