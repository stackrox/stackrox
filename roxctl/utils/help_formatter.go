package utils

import (
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/term"
)

// cobraWriter implements StringWriter using cobra.command.Print functions.
type cobraWriter cobra.Command

var _ io.StringWriter = (*cobraWriter)(nil)

func (cw *cobraWriter) WriteString(s string) (int, error) {
	(*cobra.Command)(cw).Print(s)
	return len(s), nil
}

// FormatHelp formats command help.
// FTR, kubectl uses a similar, but even more sophisticated custom formatter:
// https://github.com/kubernetes/kubectl/blob/master/pkg/util/templates/templater.go
func FormatHelp(c *cobra.Command, _ []string) {

	termWidth, _, err := term.GetSize(int(os.Stderr.Fd())) //nolint:forbidigo // TODO(ROX-13473)
	if err != nil {
		termWidth = 80
	}
	w := makeHelpWriter(
		makeFormattingWriter((*cobraWriter)(c), termWidth, defaultTabWidth))
	if len(c.Short) > 0 {
		w.WriteLn(c.Short)
	}
	if len(c.Long) > 0 {
		w.EmptyLineSeparator()
		w.WriteLn(c.Long)
	}
	if len(c.Aliases) > 0 {
		w.EmptyLineSeparator()
		w.WriteLn("Aliases:")
		w.Indent(2).WriteLn(c.NameAndAliases())
	}
	if c.HasExample() {
		w.EmptyLineSeparator()
		w.WriteLn("Examples:")
		w.Indent(2).WriteLn(c.Example)
	}
	if c.HasAvailableSubCommands() {
		if len(c.Groups()) == 0 {
			w.EmptyLineSeparator()
			w.WriteLn("Available Commands:")
			formatCommands(c.Commands(), w, "")
		} else {
			for _, group := range c.Groups() {
				w.EmptyLineSeparator()
				w.WriteLn(group.Title)
				formatCommands(c.Commands(), w, group.ID)
			}
			if !c.AllChildCommandsHaveGroup() {
				w.EmptyLineSeparator()
				w.WriteLn("Additional Commands:")
				formatCommands(c.Commands(), w, "")
			}
		}
	}
	if c.HasAvailableLocalFlags() {
		w.EmptyLineSeparator()
		w.WriteLn("Options:")
		c.LocalFlags().VisitAll(makeFlagVisitor(w))
	}
	if c.HasAvailableInheritedFlags() {
		w.EmptyLineSeparator()
		w.WriteLn("Global Options:")
		c.InheritedFlags().VisitAll(makeFlagVisitor(w))
	}
	if c.Runnable() || c.HasAvailableSubCommands() {
		w.EmptyLineSeparator()
		w.WriteLn("Usage:")
		if c.Runnable() {
			w.Indent(2).WriteLn(c.UseLine())
		}
		if c.HasAvailableSubCommands() {
			w.Indent(2).WriteLn(c.CommandPath(), " [command]")
		}
	}
	if len(c.Deprecated) != 0 {
		w.EmptyLineSeparator()
		w.WriteLn("DEPRECATED: ", c.Deprecated)
	}
}

// formatCommands prints the command name and description.
func formatCommands(commands []*cobra.Command, w *helpWriter, group string) {
	maxCommandLength := 0
	for _, command := range commands {
		if !command.IsAvailableCommand() {
			continue
		}
		maxCommandLength = max(maxCommandLength, len(command.Name()))
	}
	maxCommandLength += 5
	for _, command := range commands {
		if !command.IsAvailableCommand() || command.GroupID != group {
			continue
		}
		help := command.Short
		if len(help) == 0 {
			help = command.Long
		}
		name := command.Name()
		w.Indent(2).Write(name)
		w.Indent(-maxCommandLength).WriteLn(help)
	}
}

//nolint:errcheck,gosec
func formatFlag(sb io.StringWriter, f *pflag.Flag) {
	if len(f.Shorthand) > 0 {
		sb.WriteString("-")
		sb.WriteString(f.Shorthand)
		sb.WriteString(", ")
	}
	sb.WriteString("--")
	sb.WriteString(f.Name)
	if len(f.DefValue) > 0 {
		sb.WriteString("=")
		vt := f.Value.Type()
		if vt == "string" || vt == "duration" {
			sb.WriteString("'")
			sb.WriteString(f.DefValue)
			sb.WriteString("'")
		} else {
			sb.WriteString(f.DefValue)
		}
	}
	sb.WriteString(":\n")
}

func makeFlagVisitor(w *helpWriter) func(f *pflag.Flag) {
	firstFlag := true
	return func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		if !firstFlag {
			w.EmptyLineSeparator()
		}
		firstFlag = false
		sb := &strings.Builder{}
		formatFlag(sb, f)
		w.Indent(4, 8).WriteLn(sb.String(), f.Usage)
	}
}
