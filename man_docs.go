// Copyright 2015 Red Hat Inc. All rights reserved.
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

package cobra

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	mangen "github.com/cpuguy83/go-md2man/md2man"
	"github.com/spf13/pflag"
)

func GenManTree(cmd *Command, projectName, dir string) {
	cmd.GenManTree(projectName, dir)
}

func (cmd *Command) GenManTree(projectName, dir string) {
	for _, c := range cmd.Commands() {
		if len(c.Deprecated) != 0 || c == cmd.helpCommand {
			continue
		}
		GenManTree(c, projectName, dir)
	}
	out := new(bytes.Buffer)

	cmd.GenMan(projectName, out)

	filename := cmd.CommandPath()
	filename = dir + strings.Replace(filename, " ", "-", -1) + ".1"
	outFile, err := os.Create(filename)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer outFile.Close()
	_, err = outFile.Write(out.Bytes())
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func GenMan(cmd *Command, projectName string, out *bytes.Buffer) {
	cmd.GenMan(projectName, out)
}

func (cmd *Command) GenMan(projectName string, out *bytes.Buffer) {

	buf := genMarkdown(cmd, projectName)
	final := mangen.Render(buf)
	out.Write(final)
}

func manPreamble(out *bytes.Buffer, projectName, name, short, long string) {
	fmt.Fprintf(out, `%% %s(1)
# NAME
`, projectName)
	fmt.Fprintf(out, "%s \\- %s\n\n", name, short)
	fmt.Fprintf(out, "# SYNOPSIS\n")
	fmt.Fprintf(out, "**%s** [OPTIONS]\n\n", name)
	fmt.Fprintf(out, "# DESCRIPTION\n")
	fmt.Fprintf(out, "%s\n\n", long)
}

func manPrintFlags(out *bytes.Buffer, flags *pflag.FlagSet) {
	flags.VisitAll(func(flag *pflag.Flag) {
		if len(flag.Deprecated) > 0 {
			return
		}
		format := ""
		if len(flag.Shorthand) > 0 {
			format = "**-%s**, **--%s**"
		} else {
			format = "%s**--%s**"
		}
		if len(flag.NoOptDefVal) > 0 {
			format = format + "["
		}
		if flag.Value.Type() == "string" {
			// put quotes on the value
			format = format + "=%q"
		} else {
			format = format + "=%s"
		}
		if len(flag.NoOptDefVal) > 0 {
			format = format + "]"
		}
		format = format + "\n\t%s\n\n"
		fmt.Fprintf(out, format, flag.Shorthand, flag.Name, flag.DefValue, flag.Usage)
	})
}

func manPrintOptions(out *bytes.Buffer, command *Command) {
	flags := command.NonInheritedFlags()
	if flags.HasFlags() {
		fmt.Fprintf(out, "# OPTIONS\n")
		manPrintFlags(out, flags)
		fmt.Fprintf(out, "\n")
	}
	flags = command.InheritedFlags()
	if flags.HasFlags() {
		fmt.Fprintf(out, "# OPTIONS INHERITED FROM PARENT COMMANDS\n")
		manPrintFlags(out, flags)
		fmt.Fprintf(out, "\n")
	}
}

func genMarkdown(cmd *Command, projectName string) []byte {
	// something like `rootcmd subcmd1 subcmd2`
	commandName := cmd.CommandPath()
	// something like `rootcmd-subcmd1-subcmd2`
	dashCommandName := strings.Replace(commandName, " ", "-", -1)

	buf := new(bytes.Buffer)

	short := cmd.Short
	long := cmd.Long
	if len(long) == 0 {
		long = short
	}

	manPreamble(buf, projectName, commandName, short, long)
	manPrintOptions(buf, cmd)

	if len(cmd.Example) > 0 {
		fmt.Fprintf(buf, "# EXAMPLE\n")
		fmt.Fprintf(buf, "```\n%s\n```\n", cmd.Example)
	}

	if cmd.hasSeeAlso() {
		fmt.Fprintf(buf, "# SEE ALSO\n")
		if cmd.HasParent() {
			parentPath := cmd.Parent().CommandPath()
			dashParentPath := strings.Replace(parentPath, " ", "-", -1)
			fmt.Fprintf(buf, "**%s(1)**, ", dashParentPath)
		}

		children := cmd.Commands()
		sort.Sort(byName(children))
		for _, c := range children {
			if len(c.Deprecated) != 0 || c == cmd.helpCommand {
				continue
			}
			fmt.Fprintf(buf, "**%s-%s(1)**, ", dashCommandName, c.Name())
		}
		fmt.Fprintf(buf, "\n")
	}

	fmt.Fprintf(buf, "# HISTORY\n%s Auto generated by spf13/cobra\n", time.Now().UTC())
	return buf.Bytes()
}
