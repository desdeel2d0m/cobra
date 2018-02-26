package cobra

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

func TestGenZshCompletion(t *testing.T) {
	var debug bool
	var option string

	tcs := []struct {
		name                string
		root                *Command
		expectedExpressions []string
	}{
		{
			name: "simple command",
			root: func() *Command {
				r := &Command{
					Use:  "mycommand",
					Long: "My Command long description",
					Run:  emptyRun,
				}
				r.Flags().BoolVar(&debug, "debug", debug, "description")
				return r
			}(),
			expectedExpressions: []string{
				`(?s)function _mycommand {\s+_arguments \\\s+"--debug\[description\]".*--help.*}`,
				"#compdef _mycommand mycommand",
			},
		},
		{
			name: "flags with both long and short flags",
			root: func() *Command {
				r := &Command{
					Use:  "testcmd",
					Long: "long description",
					Run:  emptyRun,
				}
				r.Flags().BoolVarP(&debug, "debug", "d", debug, "debug description")
				return r
			}(),
			expectedExpressions: []string{
				`"\(-d --debug\)"{-d,--debug}"\[debug description\]"`,
			},
		},
		{
			name: "command with subcommands",
			root: func() *Command {
				r := &Command{
					Use:  "rootcmd",
					Long: "Long rootcmd description",
				}
				d := &Command{
					Use:   "subcmd1",
					Short: "Subcmd1 short descrition",
					Run:   emptyRun,
				}
				e := &Command{
					Use:  "subcmd2",
					Long: "Subcmd2 short description",
					Run:  emptyRun,
				}
				r.PersistentFlags().BoolVar(&debug, "debug", debug, "description")
				d.Flags().StringVarP(&option, "option", "o", option, "option description")
				r.AddCommand(d, e)
				return r
			}(),
			expectedExpressions: []string{
				`commands=\(\n\s+"help:.*\n\s+"subcmd1:.*\n\s+"subcmd2:.*\n\s+\)`,
				`_arguments \\\n.*"--debug\[description]"`,
				`_arguments -C \\\n.*"--debug\[description]"`,
				`function _rootcmd_subcmd1 {`,
				`function _rootcmd_subcmd1 {`,
				`_arguments \\\n.*"\(-o --option\)"{-o,--option}"\[option description]" \\\n`,
			},
		},
		{
			name: "filename completion",
			root: func() *Command {
				var file string
				r := &Command{
					Use:   "mycmd",
					Short: "my command short description",
					Run:   emptyRun,
				}
				r.Flags().StringVarP(&file, "config", "c", file, "config file")
				r.MarkFlagFilename("config", "ext")
				return r
			}(),
			expectedExpressions: []string{
				`\n +"\(-c --config\)"{-c,--config}"\[config file]:filename:_files"`,
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tc.root.Execute()
			buf := new(bytes.Buffer)
			tc.root.GenZshCompletion(buf)
			output := buf.Bytes()

			for _, expr := range tc.expectedExpressions {
				rgx, err := regexp.Compile(expr)
				if err != nil {
					t.Errorf("error compiling expression (%s): %v", expr, err)
				}
				if !rgx.Match(output) {
					t.Errorf("expeced completion (%s) to match '%s'", buf.String(), expr)
				}
			}
		})
	}
}

func TestGenZshCompletionHidden(t *testing.T) {
	tcs := []struct {
		name                string
		root                *Command
		expectedExpressions []string
	}{
		{
			name: "hidden commmands",
			root: func() *Command {
				r := &Command{
					Use:   "main",
					Short: "main short description",
				}
				s1 := &Command{
					Use:    "sub1",
					Hidden: true,
					Run:    emptyRun,
				}
				s2 := &Command{
					Use:   "sub2",
					Short: "short sub2 description",
					Run:   emptyRun,
				}
				r.AddCommand(s1, s2)

				return r
			}(),
			expectedExpressions: []string{
				"sub1",
			},
		},
		{
			name: "hidden flags",
			root: func() *Command {
				var hidden string
				r := &Command{
					Use:   "root",
					Short: "root short description",
					Run:   emptyRun,
				}
				r.Flags().StringVarP(&hidden, "hidden", "H", hidden, "hidden usage")
				if err := r.Flags().MarkHidden("hidden"); err != nil {
					t.Errorf("Error setting flag hidden: %v\n", err)
				}
				return r
			}(),
			expectedExpressions: []string{
				"--hidden",
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tc.root.Execute()
			buf := new(bytes.Buffer)
			tc.root.GenZshCompletion(buf)
			output := buf.String()

			for _, expr := range tc.expectedExpressions {
				if strings.Contains(output, expr) {
					t.Errorf("Expected completion (%s) not to contain '%s' but it does", output, expr)
				}
			}
		})
	}
}

func BenchmarkMediumSizeConstruct(b *testing.B) {
	root := constructLargeCommandHeirarchy()
	// if err := root.GenZshCompletionFile("_mycmd"); err != nil {
	// 	b.Error(err)
	// }

	for i := 0; i < b.N; i++ {
		buf := new(bytes.Buffer)
		err := root.GenZshCompletion(buf)
		if err != nil {
			b.Error(err)
		}
	}
}

func TestExtractFlags(t *testing.T) {
	var debug, cmdc, cmdd bool
	c := &Command{
		Use:  "cmdC",
		Long: "Command C",
	}
	c.PersistentFlags().BoolVarP(&debug, "debug", "d", debug, "debug mode")
	c.Flags().BoolVar(&cmdc, "cmd-c", cmdc, "Command C")
	d := &Command{
		Use:  "CmdD",
		Long: "Command D",
	}
	d.Flags().BoolVar(&cmdd, "cmd-d", cmdd, "Command D")
	c.AddCommand(d)

	resC := extractFlags(c)
	resD := extractFlags(d)

	if len(resC) != 2 {
		t.Errorf("expected Command C to return 2 flags, got %d", len(resC))
	}
	if len(resD) != 2 {
		t.Errorf("expected Command D to return 2 flags, got %d", len(resD))
	}
}

func constructLargeCommandHeirarchy() *Command {
	var config, st1, st2 string
	var long, debug bool
	var in1, in2 int

	r := genTestCommand("mycmd", false)
	r.PersistentFlags().StringVarP(&config, "config", "c", config, "config usage")
	if err := r.MarkPersistentFlagFilename("config", "*"); err != nil {
		panic(err)
	}
	s1 := genTestCommand("sub1", true)
	s1.Flags().BoolVar(&long, "long", long, "long descriptin")
	s2 := genTestCommand("sub2", true)
	s2.PersistentFlags().BoolVar(&debug, "debug", debug, "debug description")
	s3 := genTestCommand("sub3", true)
	s3.Hidden = true
	s1_1 := genTestCommand("sub1sub1", true)
	s1_1.Flags().StringVar(&st1, "st1", st1, "st1 description")
	s1_1.Flags().StringVar(&st2, "st2", st2, "st2 description")
	s1_2 := genTestCommand("sub1sub2", true)
	s1_3 := genTestCommand("sub1sub3", true)
	s1_3.Flags().IntVar(&in1, "int1", in1, "int1 descriptionn")
	s1_3.Flags().IntVar(&in2, "int2", in2, "int2 descriptionn")
	s2_1 := genTestCommand("sub2sub1", true)
	s2_2 := genTestCommand("sub2sub2", true)
	s2_3 := genTestCommand("sub2sub3", true)
	s2_4 := genTestCommand("sub2sub4", true)
	s2_5 := genTestCommand("sub2sub5", true)

	s1.AddCommand(s1_1, s1_2, s1_3)
	s2.AddCommand(s2_1, s2_2, s2_3, s2_4, s2_5)
	r.AddCommand(s1, s2, s3)
	r.Execute()
	return r
}

func genTestCommand(name string, withRun bool) *Command {
	r := &Command{
		Use:   name,
		Short: name + " short description",
		Long:  "Long description for " + name,
	}
	if withRun {
		r.Run = emptyRun
	}

	return r
}
