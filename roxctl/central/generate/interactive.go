package generate

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
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/flags"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	categoryAnnotation = "category"
	groupAnnotationKey = "group"
)

var (
	orderedFlagGroupNames = []string{"central", "central-db", "scanner"}
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

func isNoInteractive(f *pflag.Flag) bool {
	noIntAnn := f.Annotations[flags.NoInteractiveKey]
	if len(noIntAnn) == 0 {
		return false
	}
	return noIntAnn[0] == "true"
}

func isOptional(f *pflag.Flag) bool {
	optAnn := f.Annotations[flags.OptionalKey]
	if len(optAnn) == 0 {
		return false
	}
	return optAnn[0] == "true"
}

func isMandatory(f *pflag.Flag) bool {
	mandAnn := f.Annotations[flags.MandatoryKey]
	if len(mandAnn) == 0 {
		return false
	}
	return mandAnn[0] == "true"
}

func isPassword(f *pflag.Flag) bool {
	optAnn := f.Annotations[flags.PasswordKey]
	if len(optAnn) == 0 {
		return false
	}
	return optAnn[0] == "true"
}

func getInteractiveUsage(f *pflag.Flag) string {
	usageAnn := f.Annotations[flags.InteractiveUsageKey]
	if len(usageAnn) == 0 || usageAnn[0] == "" {
		return f.Usage
	}
	return usageAnn[0]
}

func readUserInputFromFlag(f *pflag.Flag) (string, error) {
	usage := getInteractiveUsage(f)

	var prompt string
	if f.DefValue != "" {
		optText := ""
		if isOptional(f) {
			optText = ", optional"
		}
		prompt = fmt.Sprintf("Enter %s (default: %q%s): ", usage, f.DefValue, optText)
	} else {
		optText := ""
		if isOptional(f) {
			optText = " (optional)"
		}
		prompt = fmt.Sprintf("Enter %s%s: ", usage, optText)
	}

	var err error
	var text string
	if isPassword(f) {
		text, err = readPassword(prompt)
	} else {
		text, err = readUserInput(prompt)
	}

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
	if isMandatory(f) && s == "" {
		printlnToStderr("A value must be entered. Please try again.")
		return readUserString(f)
	}
	return s
}

func readPassword(prompt string) (string, error) {
	fd := int(os.Stdin.Fd())
	if !terminal.IsTerminal(fd) {
		printlnToStderr("%s", "Warning: Entered password will be echoed in this mode. Use 'roxctl generate central interactive' instead if you would not like the password echoed.")
	}

	printToStderr("%s", prompt)
	passwd, err := getPassword(fd)
	if err != nil {
		return "", err
	}

	// Re enter password prompt only for the roxctl case, not for docker run
	if terminal.IsTerminal(fd) && passwd != "" {
		printToStderr("Re-%s: ", strings.TrimSpace(strings.ToLower(strings.Split(prompt, "(")[0])))
		reEnteredPasswd, err := getPassword(fd)
		if err != nil {
			return "", err
		}
		if passwd != reEnteredPasswd {
			printlnToStderr("Error: Passwords do not match. Please try again.")
			return readPassword(prompt)
		}
	}
	return passwd, nil
}

func getPassword(fd int) (passwd string, err error) {
	if terminal.IsTerminal(fd) {
		bytes, err := terminal.ReadPassword(fd)
		if err != nil {
			return "", err
		}
		passwd = string(bytes)
		printlnToStderr("")
	} else {
		reader := bufio.NewReader(os.Stdin)
		passwd, err = reader.ReadString('\n')
		if err != nil {
			return "", err
		}
	}
	return strings.TrimSuffix(passwd, "\n"), nil
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

func chooseCommand(argSlice *argSlice, prompt string, c *cobra.Command) {
	for {
		cmdString, err := readUserInput(prompt)
		if err != nil {
			printlnToStderr("\nCould not read user input. Did you specify '-i' in the Docker run command?")
			os.Exit(1)
		}
		for _, subCommand := range c.Commands() {
			if subCommand.Name() == cmdString {
				walkTreeWithArgSlice(argSlice, subCommand)
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
	for i, flag := range flags {
		if flag.Name == flagName {
			foundFlag = &flags[i]
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

// optional interface to indicate boolean flags that can be
// supplied without "=value" text
// COPIED FROM COBRA.
type boolFlag interface {
	pflag.Value
	IsBoolFlag() bool
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
		iPos := sliceutils.Find(orderedFlagGroupNames, groupsSlice[i].name)
		jPos := sliceutils.Find(orderedFlagGroupNames, groupsSlice[j].name)
		// If they're both not in the list of ordered flag groups, just sort alphabetically.
		if iPos == -1 && jPos == -1 {
			return groupsSlice[i].name < groupsSlice[j].name
		}
		return iPos < jPos
	})
	return groupsSlice
}

func processFlagWraps(argSlice *argSlice, fws []flagWrap) {
	flagsByName := make(map[string]*pflag.Flag)
	for _, fw := range fws {
		flagsByName[fw.Name] = fw.Flag
	}

	for _, fw := range fws {
		if fw.Hidden || isNoInteractive(fw.Flag) || fw.Flag == nil {
			continue
		}

		// set default values for image-{main,scanner,scanner-db} flags
		if fw.Flag.Name == flags.FlagNameMainImage || fw.Flag.Name == flags.FlagNameScannerImage || fw.Flag.Name == flags.FlagNameScannerDBImage || fw.Flag.Name == flags.FlagNameCentralDBImage {
			imgDefArg := argSlice.findArgByName(flags.FlagNameImageDefaults)
			if imgDefArg == nil {
				panic(fmt.Sprintf("unable to find flag '%s'", flags.FlagNameImageDefaults))
			}
			var flavor *defaults.ImageFlavor
			if f, err := defaults.GetImageFlavorByName(imgDefArg.value, buildinfo.ReleaseBuild); err == nil {
				flavor = &f
			}
			switch fw.Flag.Name {
			case flags.FlagNameMainImage:
				if fw.Flag.DefValue == "" {
					fw.Flag.DefValue = flavor.MainImage()
				}
			case flags.FlagNameScannerImage:
				if fw.Flag.DefValue == "" {
					fw.Flag.DefValue = flavor.ScannerImage()
				}
			case flags.FlagNameScannerDBImage:
				if fw.Flag.DefValue == "" {
					fw.Flag.DefValue = flavor.ScannerDBImage()
				}
			case flags.FlagNameCentralDBImage:
				if fw.Flag.DefValue == "" {
					fw.Flag.DefValue = flavor.CentralDBImage()
				}
			}
		}

		depUnmet := false
		for _, dep := range fw.Annotations[flags.DependenciesKey] {
			flag := flagsByName[dep]
			//nolint:staticcheck // SA5011 flag is definitely not nil because utils.Must panics.
			if flag == nil {
				utils.CrashOnError(errors.Errorf("invalid flag dependency %q", dep))
			}
			//nolint:staticcheck // SA5011 flag is definitely not nil because utils.Must panics.
			if !argSlice.flagNameIsSetExplicitly(flag.Name) {
				depUnmet = true
				break
			}
		}
		if depUnmet {
			continue
		}

		processedSuccessfully := false
		for !processedSuccessfully {
			value, commandline := processFlag(fw.Flag)

			// validate value of image-defaults flag
			if fw.Flag.Name == flags.FlagNameImageDefaults {
				if _, err := defaults.GetImageFlavorByName(value, buildinfo.ReleaseBuild); value != "" && err != nil {
					printlnToStderr(err.Error())
					continue
				}
			}

			// Verify flag parsing
			if value != "" {
				if err := fw.Value.Set(value); err != nil {
					printlnToStderr(err.Error())
					continue
				}
			}
			// For bool flags, we want to convert all values to "true" or "false"
			if _, ok := fw.Flag.Value.(boolFlag); ok {
				value = fw.Flag.Value.String()
			}
			processedSuccessfully = true
			argSlice.addArg(arg{commandLine: commandline, value: value, flagName: fw.Name})
			if childFlags, exists := fw.childFlags[value]; exists {
				processFlagWraps(argSlice, childFlags)
			}
		}
	}
}

type argSlice struct {
	args []arg
}

func (a *argSlice) findArgByName(name string) *arg {
	for _, f := range a.args {
		if f.flagName == name {
			return &f
		}
	}
	return nil
}

func (a *argSlice) addArg(arg arg) {
	a.args = append(a.args, arg)
}

func (a *argSlice) flagNameIsSetExplicitly(flagName string) bool {
	for _, arg := range a.args {
		if arg.commandLine != "" && arg.flagName == flagName {
			return true
		}
	}
	return false
}

type arg struct {
	commandLine string
	value       string
	flagName    string
}

func walkTree(c *cobra.Command) []string {
	argSlice := argSlice{}
	walkTreeWithArgSlice(&argSlice, c)

	var args []string
	for _, arg := range argSlice.args {
		if arg.commandLine != "" {
			args = append(args, arg.commandLine)
		}
	}
	return args
}

func walkTreeWithArgSlice(argSlice *argSlice, c *cobra.Command) {
	argSlice.addArg(arg{commandLine: c.Name()})

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
		processFlagWraps(argSlice, fg.flags)
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
		chooseCommand(argSlice, cmdPrompt, c)
	}
}
