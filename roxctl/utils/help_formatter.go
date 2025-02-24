package utils

import (
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type cobraWriter struct {
	c *cobra.Command
}

var _ io.StringWriter = (*cobraWriter)(nil)

func (cw *cobraWriter) WriteString(s string) (int, error) {
	cw.c.Print(s)
	return len(s), nil
}

func (cw *cobraWriter) Println(s ...any) {
	cw.c.Println(s...)
}

type indentWriter struct {
	w        io.StringWriter
	maxWidth int
}

func (iw *indentWriter) Write(s string, indent ...int) {
	_, _ = makeWriter(iw.w, iw.maxWidth, indent...).WriteString(s)
}

func (iw *indentWriter) WriteLn(s string, indent ...int) {
	if len(s) > 0 {
		iw.Write(s, indent...)
	}
	iw.NewLine()
}

func (iw *indentWriter) NewLine() {
	_, _ = iw.w.WriteString("\n")
}

func makeFlagVisitor(iw *indentWriter) func(f *pflag.Flag) {
	return func(f *pflag.Flag) {
		if f.Hidden {
			return
		}
		sb := makeWriter(iw.w, iw.maxWidth, 4, 8)
		formatFlag(sb, f)
		iw.NewLine()
		iw.NewLine()
	}
}

// FormatHelp formats command help.
// FTR, kubectl uses a similar, but even more sophisticated custom formatter:
// https://github.com/kubernetes/kubectl/blob/master/pkg/util/templates/templater.go
func FormatHelp(c *cobra.Command, _ []string) {
	iw := &indentWriter{
		&cobraWriter{c},
		80,
	}
	if len(c.Short) > 0 {
		iw.WriteLn(c.Short)
	}
	if len(c.Long) > 0 {
		if len(c.Short) > 0 {
			iw.NewLine()
		}
		iw.WriteLn(c.Long)
	}
	if len(c.Aliases) > 0 {
		iw.NewLine()
		iw.WriteLn("Aliases:")
		iw.WriteLn(c.NameAndAliases(), 2)
	}
	if c.HasExample() {
		iw.NewLine()
		iw.WriteLn("Examples:")
		iw.WriteLn(c.Example, 2)
	}
	if c.HasAvailableSubCommands() {
		if len(c.Groups()) == 0 {
			iw.NewLine()
			iw.WriteLn("Available Commands:")
			formatCommands(c.Commands(), iw, "")
		} else {
			for _, group := range c.Groups() {
				iw.NewLine()
				iw.WriteLn(group.Title)
				formatCommands(c.Commands(), iw, group.ID)
			}
			if !c.AllChildCommandsHaveGroup() {
				iw.NewLine()
				iw.WriteLn("Additional Commands:")
				formatCommands(c.Commands(), iw, "")
			}
		}
	}
	visitor := makeFlagVisitor(iw)
	hasFlags := false
	if c.HasAvailableLocalFlags() {
		iw.NewLine()
		iw.WriteLn("Options:")
		c.LocalFlags().VisitAll(visitor)
		hasFlags = true
	}
	if c.HasAvailableInheritedFlags() {
		if !hasFlags {
			iw.NewLine()
		}
		iw.WriteLn("Global Options:")
		c.InheritedFlags().VisitAll(visitor)
	}
	iw.WriteLn("Usage:")
	if c.Runnable() {
		iw.WriteLn(c.UseLine(), 2)
	}
	if c.HasAvailableSubCommands() {
		iw.Write(c.CommandPath(), 2)
		iw.WriteLn(" [command]")
	}

}

// formatCommands prints the command name and description.
func formatCommands(commands []*cobra.Command, iw *indentWriter, group string) {
	padding := 0
	for _, command := range commands {
		if !command.IsAvailableCommand() {
			continue
		}
		padding = max(padding, len(command.Name()))
	}
	const leftPadding, interPadding = 2, 3
	for _, command := range commands {
		if !command.IsAvailableCommand() || command.GroupID != group {
			continue
		}
		name := command.Name()
		iw.Write(name, leftPadding)
		help := command.Short
		if len(help) == 0 {
			help = command.Long
		}
		iw.WriteLn(help, padding-len(name)+interPadding, padding+leftPadding+interPadding)
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
	sb.WriteString(f.Usage)
}
