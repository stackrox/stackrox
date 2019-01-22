package deploy

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stackrox/rox/pkg/docker"
)

const (
	categoryAnnotation = "category"
	groupAnnotation    = "group"

	subgroupAnnotation = "subgroup"
)

func getFlagGroupMap() map[string]*flagGroup {
	return map[string]*flagGroup{
		"central": {
			name:       "central",
			optional:   false,
			groupOrder: 0,
		},
		"clairify": {
			name:       "clairify",
			optional:   false,
			groupOrder: 1,
		},
		"monitoring": {
			name:        "monitoring",
			optional:    true,
			groupOrder:  2,
			groupPrompt: "Would you like to run the monitoring stack?",
			cmdLineSpec: "--monitoring-type=none",

			subgroupPrompt: "Enter persistence type for monitoring (hostpath, pvc, none):",
			subgroup: map[string][]*pflag.Flag{
				"none": {},
			},
			subgroupCmdLineSpecTemplate: "--monitoring-persistence-type=%s",
		},
	}
}

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
	prompt += " [y/N] "
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
		printlnToStderr("Error reading value from command line. Please try again.")
		return readUserString(f)
	}
	return s
}

func printlnToStderr(t string, args ...interface{}) {
	printToStderr(t+"\n", args...)
}

func printToStderr(t string, args ...interface{}) {
	str := fmt.Sprintf(t, args...)
	if str != "" {
		r, n := utf8.DecodeRuneInString(str)
		str = string(unicode.ToUpper(r)) + str[n:]
	}
	fmt.Fprint(os.Stderr, str)
}

func processFlag(f *pflag.Flag) (string, string) {
	userInput := readUserString(f)
	if userInput == "" {
		return "", ""
	}
	return userInput, fmt.Sprintf("--%s=%s", f.Name, userInput)
}

func choseCommand(prompt string, c *cobra.Command) (args []string) {
	for true {
		cmdString, err := readUserInput(prompt)
		if err != nil {
			printlnToStderr("\nCould not read user input. Did you specify '-i' in the Docker run command?")
			os.Exit(1)
		}
		for _, subCommand := range c.Commands() {
			if subCommand.Name() == cmdString {
				args = append(args, walkTree(subCommand)...)
				return
			}
		}
		printlnToStderr("'%s' is not a valid option. Please try again.", cmdString)
	}
	return
}

type flagGroup struct {
	name        string
	optional    bool
	groupOrder  int
	groupPrompt string
	flags       []*pflag.Flag
	cmdLineSpec string

	subgroupPrompt              string
	subgroup                    map[string][]*pflag.Flag
	subgroupCmdLineSpecTemplate string
}

func getFirstFromStringSliceOrEmpty(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}

func flagGroups(flags []*pflag.Flag) []*flagGroup {
	groups := make(map[string]*flagGroup)
	// Iterate over the flags to get the groups
	flagGroupMap := getFlagGroupMap()
	for _, f := range flags {
		name := getFirstFromStringSliceOrEmpty(f.Annotations[groupAnnotation])
		// Check global flag group
		group, ok := flagGroupMap[name]
		if !ok {
			var ok bool
			// Check per function group
			if group, ok = groups[name]; !ok {
				group = &flagGroup{}
			}
		}

		if _, ok := f.Annotations[subgroupAnnotation]; !ok {
			groups[name] = group
			group.flags = append(group.flags, f)
		}
	}
	// Iterate over the flags against to attach the subgroup
	for _, f := range flags {
		subgroup := getFirstFromStringSliceOrEmpty(f.Annotations[subgroupAnnotation])
		if subgroup == "" {
			continue
		}
		name := getFirstFromStringSliceOrEmpty(f.Annotations[groupAnnotation])
		if name == "" {
			panic(fmt.Sprintf("Invalid annotations on flags. Flag %s with subgroup %s must have a valid group assigned", f.Name, subgroup))
		}
		group := groups[name]
		group.subgroup[subgroup] = append(group.subgroup[subgroup], f)
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
		if f.Hidden {
			return
		}
		allFlags = append(allFlags, f)
	})
	c.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
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
				if fg.cmdLineSpec != "" {
					args = append(args, fg.cmdLineSpec)
				}
				continue
			}
		}
		for _, flag := range fg.flags {
			if flag.Hidden {
				continue
			}
			for {
				if value, commandline := processFlag(flag); flag.NoOptDefVal == "" {
					// Verify flag parsing
					if err := flag.Value.Set(value); err != nil {
						printlnToStderr(err.Error())
						continue
					}
					args = append(args, commandline)
				}
				break
			}
		}
		if fg.subgroup != nil {
			input, err := readUserInput(fg.subgroupPrompt)
			if err != nil {
				if docker.IsContainerized() {
					printToStderr("\nCould not read user input. Did you specify '-i' in the Docker run command?\n")
				} else {
					printToStderr("\nError reading user input: %v", err)
				}
				os.Exit(1)
			}
			var subgroupFlags []*pflag.Flag
			for {
				var ok bool
				subgroupFlags, ok = fg.subgroup[input]
				if ok {
					// Currently the only subgroup is monitoring persistence
					if fg.subgroupCmdLineSpecTemplate != "" {
						args = append(args, fmt.Sprintf(fg.subgroupCmdLineSpecTemplate, input))
					}
					break
				}
				printToStderr(fmt.Sprintf("\n%q is not a valid option", input))
				input, err = readUserInput(fg.subgroupPrompt)
				if err != nil {
					if docker.IsContainerized() {
						printToStderr("\nCould not read user input. Did you specify '-i' in the Docker run command?\n")
					} else {
						printToStderr("\nError reading user input: %v", err)
					}
					os.Exit(1)
				}
			}
			for _, f := range subgroupFlags {
				if val, flag := processFlag(f); val != "" {
					args = append(args, flag)
				}
			}
		}
	}

	// group commands by their annotation categories
	categoriesToCommands := make(map[string][]string)
	for _, cmd := range c.Commands() {
		if cmd.Hidden {
			continue
		}
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
