// Copyright © 2013 Steve Francia <spf@spf13.com>.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Commands similar to git, go tools and other modern CLI tools
// inspired by go, go-Commander, gh and subcommand

package cobra

import (
	"bytes"
	"fmt"
	flag "github.com/spf13/pflag"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"text/template"
)

var _ = flag.ContinueOnError

// A Commander holds the configuration for the command line tool.
type Commander struct {
	// A Commander is also a Command for top level and global help & flags
	Command

	args          []string
	output        io.Writer            // nil means stderr; use out() accessor
	UsageFunc     func(*Command) error // Usage can be defined by application
	UsageTemplate string               // Can be defined by Application
	HelpTemplate  string               // Can be defined by Application
}

// Provide the user with a new commander.
func NewCommander() (c *Commander) {
	c = new(Commander)
	c.cmdr = c
	c.UsageFunc = c.defaultUsage
	c.initTemplates()
	return
}

// Name for commander, should match application name
func (c *Commander) SetName(name string) {
	c.name = name
}

// os.Args[1:] by default, if desired, can be overridden
// particularly useful when testing.
func (c *Commander) SetArgs(a []string) {
	c.args = a
}

// Call execute to use the args (os.Args[1:] by default)
// and run through the command tree finding appropriate matches
// for commands and then corresponding flags.
func (c *Commander) Execute() (err error) {
	if len(c.args) == 0 {
		err = c.execute(os.Args[1:])
	} else {
		err = c.execute(c.args)
	}
	return
}

func (c *Commander) out() io.Writer {
	if c.output == nil {
		return os.Stderr
	}
	return c.output
}

func (cmdr *Commander) defaultUsage(c *Command) error {
	err := tmpl(cmdr.out(), cmdr.UsageTemplate, c)
	if err != nil {
		c.Println(err)
	}
	return err
}

//Print to out
func (c *Commander) POut(i ...interface{}) {
	fmt.Fprint(c.out(), i...)
}

// SetOutput sets the destination for usage and error messages.
// If output is nil, os.Stderr is used.
func (c *Commander) SetOutput(output io.Writer) {
	c.output = output
}

func (c *Commander) initTemplates() {
	c.UsageTemplate = `{{ $cmd := . }}{{.CommandPath | printf "%-11s"}} :: {{.Short}}
Usage:
    {{.UseLine}}{{if .HasSubCommands}} command{{end}}{{if .HasFlags}} [flags]{{end}}
{{ if .HasSubCommands}}
The commands are:
{{range .Commands}}{{if .Runnable}}
    {{.UseLine | printf "%-11s"}} {{.Short}}{{end}}{{end}}
Use "{{$.CommandPath}} help [command]" for more information about a command.
{{end}}
Additional help topics: {{if gt .Commands 0 }}
    {{range .Commands}}{{if not .Runnable}}{{.CommandPath | printf "%-11s"}} {{.Short}}{{end}}{{end}}{{end}}{{if gt .Parent.Commands 1 }}
    {{range .Parent.Commands}}{{if .Runnable}}{{if not (eq .Name $cmd.Name) }}{{end}}
    {{.CommandPath | printf "%-11s"}} :: {{.Short}}{{end}}{{end}}{{end}}
    Use "{{.Commander.Name}} help [topic]" for more information about that topic.
`

	c.HelpTemplate = `{{if .Runnable}}Usage: {{.ProgramName}} {{.UsageLine}}

{{end}}{{.Long | trim}}
`
}

// Command is just that, a command for your application.
// eg.  'go run' ... 'run' is the command. Cobra requires
// you to define the usage and description as part of your command
// definition to ensure usability.
type Command struct {
	// Name is the command name, usually the executable's name.
	name string
	// The one-line usage message.
	Use string
	// The short description shown in the 'help' output.
	Short string
	// The long message shown in the 'help <this-command>' output.
	Long string
	// Set of flags specific to this command.
	flags *flag.FlagSet
	// Set of flags children commands will inherit
	pflags *flag.FlagSet
	// Run runs the command.
	// The args are the arguments after the command name.
	Run func(cmd *Command, args []string)
	// Commands is the list of commands supported by this Commander program.
	commands []*Command
	// Parent Command for this command
	parent *Command
	// Commander
	cmdr         *Commander
	flagErrorBuf *bytes.Buffer
}

// find the target command given the args and command tree
func (c *Command) Find(args []string) (cmd *Command, a []string, err error) {
	if c == nil {
		return nil, nil, fmt.Errorf("Called find() on a nil Command")
	}

	validSubCommand := false
	if len(args) > 1 && c.HasSubCommands() {
		for _, cmd := range c.commands {
			if cmd.Name() == args[0] {
				validSubCommand = true
				return cmd.Find(args[1:])
			}
		}
	}
	if !validSubCommand && c.Runnable() {
		return c, args, nil
	}

	return nil, nil, nil
}

func (c *Command) Commander() *Commander {
	return c.cmdr
}

// execute the command determined by args and the command tree
func (c *Command) execute(args []string) (err error) {
	err = fmt.Errorf("unknown subcommand %q\nRun 'help' for usage.\n", args[0])

	if c == nil {
		return fmt.Errorf("Called Execute() on a nil Command")
	}

	cmd, a, e := c.Find(args)
	if e == nil {
		err = cmd.ParseFlags(a)
		if err != nil {
			cmd.Usage()
			return err
		} else {
			argWoFlags := cmd.Flags().Args()
			cmd.Run(cmd, argWoFlags)
			return nil
		}
	}
	err = e
	return err
}

// Used for testing
func (c *Command) ResetCommands() {
	c.commands = nil
}

func (c *Command) Commands() []*Command {
	return c.commands
}

// Add one or many commands as children of this
func (c *Command) AddCommand(cmds ...*Command) {
	for i, x := range cmds {
		if cmds[i] == c {
			panic("Command can't be a child of itself")
		}
		cmds[i].parent = c
		cmds[i].cmdr = cmds[i].parent.cmdr
		c.commands = append(c.commands, x)
	}
}

func (c *Command) Print(i ...interface{}) {
	c.cmdr.POut(i...)
}

func (c *Command) Println(i ...interface{}) {
	str := fmt.Sprintln(i...)
	c.Print(str)
}

func (c *Command) Printf(format string, i ...interface{}) {
	str := fmt.Sprintf(format, i...)
	c.Print(str)
}

func (c *Command) Usage() error {
	err := c.cmdr.UsageFunc(c)
	if err != nil {
		fmt.Println(err)
	}

	return err
}

func (c *Command) CommandPath() string {
	str := c.Name()
	x := c
	for x.HasParent() {
		str = x.parent.Name() + " " + str
		x = x.parent
	}

	return str
}

//The full usage for a given command (including parents)
func (c *Command) UseLine() string {
	str := ""
	if c.HasParent() {
		str = c.parent.CommandPath() + " "
	}
	return str + c.Use
}

// For use in determining which flags have been assigned to which commands
// and which persist
func (c *Command) DebugFlags() {
	c.Println("DebugFlags called on", c.Name())
	var debugflags func(*Command)

	debugflags = func(x *Command) {
		if x.HasFlags() || x.HasPersistentFlags() {
			c.Println(x.Name())
		}
		if x.HasFlags() {
			x.flags.VisitAll(func(f *flag.Flag) {
				if x.HasPersistentFlags() {
					if x.persistentFlag(f.Name) == nil {
						c.Println("  -"+f.Shorthand+",", "--"+f.Name, "["+f.DefValue+"]", "", f.Value, "  [L]")
					} else {
						c.Println("  -"+f.Shorthand+",", "--"+f.Name, "["+f.DefValue+"]", "", f.Value, "  [LP]")
					}
				} else {
					c.Println("  -"+f.Shorthand+",", "--"+f.Name, "["+f.DefValue+"]", "", f.Value, "  [L]")
				}
			})
		}
		if x.HasPersistentFlags() {
			x.pflags.VisitAll(func(f *flag.Flag) {
				if x.HasFlags() {
					if x.flags.Lookup(f.Name) == nil {
						c.Println("  -"+f.Shorthand+",", "--"+f.Name, "["+f.DefValue+"]", "", f.Value, "  [P]")
					}
				} else {
					c.Println("  -"+f.Shorthand+",", "--"+f.Name, "["+f.DefValue+"]", "", f.Value, "  [P]")
				}
			})
		}
		c.Println(x.flagErrorBuf)
		if x.HasSubCommands() {
			for _, y := range x.commands {
				debugflags(y)
			}
		}
	}

	debugflags(c)
}

// Usage prints the usage details to the standard output.
//func (c *Command) PrintUsage() {
//if c.Runnable() {
//c.Printf("usage: %s\n\n", c.Usage())
//}

//c.Println(strings.Trim(c.Long, "\n"))
//}

// Name returns the command's name: the first word in the use line.
func (c *Command) Name() string {
	if c.name != "" {
		return c.name
	}
	name := c.Use
	i := strings.Index(name, " ")
	if i >= 0 {
		name = name[:i]
	}
	return name
}

// Determine if the command is itself runnable
func (c *Command) Runnable() bool {
	return c.Run != nil
}

// Determine if the command has children commands
func (c *Command) HasSubCommands() bool {
	return len(c.commands) > 0
}

// Determine if the command is a child command
func (c *Command) HasParent() bool {
	return c.parent != nil
}

// Get the Commands FlagSet
func (c *Command) Flags() *flag.FlagSet {
	if c.flags == nil {
		c.flags = flag.NewFlagSet(c.Name(), flag.ContinueOnError)
		if c.flagErrorBuf == nil {
			c.flagErrorBuf = new(bytes.Buffer)
		}
		c.flags.SetOutput(c.flagErrorBuf)
	}
	return c.flags
}

// Get the Commands Persistent FlagSet
func (c *Command) PersistentFlags() *flag.FlagSet {
	if c.pflags == nil {
		c.pflags = flag.NewFlagSet(c.Name(), flag.ContinueOnError)
		if c.flagErrorBuf == nil {
			c.flagErrorBuf = new(bytes.Buffer)
		}
		c.pflags.SetOutput(c.flagErrorBuf)
	}
	return c.pflags
}

// Intended for use in testing
func (c *Command) ResetFlags() {
	c.flagErrorBuf = new(bytes.Buffer)
	c.flags = flag.NewFlagSet(c.Name(), flag.ContinueOnError)
	c.flags.SetOutput(c.flagErrorBuf)
	c.pflags = flag.NewFlagSet(c.Name(), flag.ContinueOnError)
	c.pflags.SetOutput(c.flagErrorBuf)
}

// Does the command contain flags (local not persistent)
func (c *Command) HasFlags() bool {
	return c.Flags().HasFlags()
}

// Does the command contain persistent flags
func (c *Command) HasPersistentFlags() bool {
	return c.PersistentFlags().HasFlags()
}

// Climbs up the command tree looking for matching flag
func (c *Command) Flag(name string) (flag *flag.Flag) {
	flag = c.Flags().Lookup(name)

	if flag == nil {
		flag = c.persistentFlag(name)
	}

	return
}

// recursively find matching persistent flag
func (c *Command) persistentFlag(name string) (flag *flag.Flag) {
	if c.HasPersistentFlags() {
		flag = c.PersistentFlags().Lookup(name)
	}

	if flag == nil && c.HasParent() {
		flag = c.parent.persistentFlag(name)
	}
	return
}

// Parses persistent flag tree & local flags
func (c *Command) ParseFlags(args []string) (err error) {
	c.mergePersistentFlags()

	err = c.Flags().Parse(args)
	if err != nil {
		return err
	}
	return nil
}

func (c *Command) mergePersistentFlags() {
	var rmerge func(x *Command)

	rmerge = func(x *Command) {
		if x.HasPersistentFlags() {
			x.PersistentFlags().VisitAll(func(f *flag.Flag) {
				if c.Flags().Lookup(f.Name) == nil {
					c.Flags().AddFlag(f)
				}
			})
		}
		if x.HasParent() {
			rmerge(x.parent)
		}
	}

	rmerge(c)
}

func (c *Command) Parent() *Command {
	return c.parent
}

func Gt(a interface{}, b interface{}) bool {
	var left, right int64
	av := reflect.ValueOf(a)

	switch av.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		left = int64(av.Len())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		left = av.Int()
	case reflect.String:
		left, _ = strconv.ParseInt(av.String(), 10, 64)
	}

	bv := reflect.ValueOf(b)

	switch bv.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		right = int64(bv.Len())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		right = bv.Int()
	case reflect.String:
		right, _ = strconv.ParseInt(bv.String(), 10, 64)
	}

	return left > right
}

func Eq(a interface{}, b interface{}) bool {
	av := reflect.ValueOf(a)
	bv := reflect.ValueOf(b)

	switch av.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice:
		panic("Eq called on unsupported type")
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return av.Int() == bv.Int()
	case reflect.String:
		return av.String() == bv.String()
	}
	return false
}

// tmpl executes the given template text on data, writing the result to w.
func tmpl(w io.Writer, text string, data interface{}) error {
	t := template.New("top")
	t.Funcs(template.FuncMap{
		"trim": strings.TrimSpace,
		"gt":   Gt,
		"eq":   Eq,
	})
	template.Must(t.Parse(text))
	return t.Execute(w, data)
}
