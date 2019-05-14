package deploy

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	categoryAnnotation = "category"
	groupAnnotationKey = "group"
)

var (
	orderedFlagGroupNames = []string{"central", "scanner", "monitoring"}
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

func isOptional(f *pflag.Flag) bool {
	optAnn := f.Annotations["optional"]
	if len(optAnn) == 0 {
		return false
	}
	return optAnn[0] == "true"
}

func readUserInputFromFlag(f *pflag.Flag) (string, error) {
	var prompt string
	if f.Value.String() != "" {
		optText := ""
		if isOptional(f) {
			optText = ", optional"
		}
		prompt = fmt.Sprintf("Enter %s (default: '%s'%s): ", f.Usage, f.Value, optText)
	} else {
		optText := ""
		if isOptional(f) {
			optText = " (optional)"
		}
		prompt = fmt.Sprintf("Enter %s%s: ", f.Usage, optText)
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
	for {
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
}

type flagWrap struct {
	*pflag.Flag
	childFlags map[string][]flagWrap
}

func addChild(child *pflag.Flag, flags []flagWrap, path []string) {
	flagName, flagValue := parseKeyValueAnnotation(path[0])
	var foundFlag *flagWrap
	for _, flag := range flags {
		if flag.Name == flagName {
			foundFlag = &flag
			break
		}
	}
	if foundFlag == nil {
		panic(fmt.Sprintf("Couldn't find flag matching annotation: %+v", path))
	}
	if len(path) > 1 {
		addChild(child, foundFlag.childFlags[flagValue], path[1:])
		return
	}
	foundFlag.childFlags[flagValue] = append(foundFlag.childFlags[flagValue], wrapFlag(child))
}

func wrapFlag(flag *pflag.Flag) flagWrap {
	return flagWrap{
		Flag:       flag,
		childFlags: make(map[string][]flagWrap),
	}
}

type flagGroup struct {
	name  string
	flags []flagWrap
}

func parseKeyValueAnnotation(annotation string) (flagName, flagValue string) {
	splitString := strings.Split(annotation, "=")
	return splitString[0], splitString[1]
}

func (f *flagGroup) addFlag(flag *pflag.Flag) {
	if annotations := flag.Annotations[groupAnnotationKey]; len(annotations) > 1 {
		addChild(flag, f.flags, annotations[1:])
		return
	}
	f.flags = append(f.flags, wrapFlag(flag))
}

func getOrCreateGroup(groups map[string]*flagGroup, groupAnnotation []string) *flagGroup {
	var rootGroupName string
	if len(groupAnnotation) > 0 {
		rootGroupName = groupAnnotation[0]
	}
	group, ok := groups[rootGroupName]
	if !ok {
		group = &flagGroup{name: rootGroupName}
		groups[rootGroupName] = group
	}
	return group
}

func flagGroups(flags []*pflag.Flag) []*flagGroup {
	groups := make(map[string]*flagGroup)
	sort.Slice(flags, func(i, j int) bool {
		return len(flags[i].Annotations[groupAnnotationKey]) < len(flags[j].Annotations[groupAnnotationKey])
	})
	for _, flag := range flags {
		group := getOrCreateGroup(groups, flag.Annotations[groupAnnotationKey])
		group.addFlag(flag)
	}
	groupsSlice := make([]*flagGroup, 0, len(groups))
	for _, group := range groups {
		groupsSlice = append(groupsSlice, group)
	}
	sort.Slice(groupsSlice, func(i, j int) bool {
		iPos := sliceutils.StringFind(orderedFlagGroupNames, groupsSlice[i].name)
		jPos := sliceutils.StringFind(orderedFlagGroupNames, groupsSlice[j].name)
		// If they're both not in the list of ordered flag groups, just sort alphabetically.
		if iPos == -1 && jPos == -1 {
			return groupsSlice[i].name < groupsSlice[j].name
		}
		return iPos < jPos
	})
	return groupsSlice
}

func processFlagWraps(fws []flagWrap) (args []string) {
	flagsByName := make(map[string]*pflag.Flag)
	for _, fw := range fws {
		flagsByName[fw.Name] = fw.Flag
	}

	for _, fw := range fws {
		if fw.Hidden {
			continue
		}

		depUnmet := false
		for _, dep := range fw.Annotations["dependencies"] {
			flag := flagsByName[dep]
			if flag == nil {
				utils.Must(errors.Errorf("invalid flag dependency %q", dep))
			}
			if !flag.Changed {
				depUnmet = true
				break
			}
		}
		if depUnmet {
			continue
		}

		for {
			if value, commandline := processFlag(fw.Flag); fw.NoOptDefVal == "" {
				// Verify flag parsing
				if err := fw.Value.Set(value); err != nil {
					printlnToStderr(err.Error())
					continue
				}
				args = append(args, commandline)
				if childFlags, exists := fw.childFlags[value]; exists {
					args = append(args, processFlagWraps(childFlags)...)
				}
			}
			break
		}
	}
	return
}

func walkTree(c *cobra.Command) (args []string) {
	args = []string{c.Name()}

	var allFlags []*pflag.Flag
	flagAppender := func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		allFlags = append(allFlags, f)
	}
	c.PersistentFlags().VisitAll(flagAppender)
	c.Flags().VisitAll(flagAppender)

	flagGroups := flagGroups(allFlags)

	for _, fg := range flagGroups {
		args = append(args, processFlagWraps(fg.flags)...)
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
