package main

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/mattn/go-zglob"
	"io"
	"os"
)

// Exit codes are int values that represent an exit code for a particular error.
const (
	ExitCodeOK = iota
	ExitCodeError
)

var opts struct {
	Misc Misc   `group:"Miscellaneous"`
	Out  Output `group:"Output control"`
}

type Misc struct {
	Help    bool `short:"h" long:"help" description:"Show this help message and exit"`
	Version bool `short:"v" long:"version" description:"Show version information and exit"`
}

type Output struct {
	File   string `short:"f" long:"file" description:"Output file path"`
	Output string `short:"o" long:"output" description:"Output format" choice:"csv" choice:"json" choice:"table" choice:"tsv" default:"table"`
	Query  string `long:"query" description:"JMESPath query string\nSee http://jmespath.org/ for more information and examples"`
	Quiet  bool   `short:"q" long:"quiet" description:"Suppress all normal output"`
	Target string `short:"t" long:"target" description:"Target value used for aggregation" choice:"cpusec" choice:"pcpu" choice:"clocksec" choice:"pclock" default:"clocksec"`
}

// CLI is the command line object.
type CLI struct {
	// outStream and errStream are the stdout and stderr
	// to write message from the CLI.
	outStream, errStream io.Writer
}

// Run invokes the CLI with the given arguments.
func (cli *CLI) Run(args []string) int {
	parser := flags.NewParser(&opts, flags.PrintErrors|flags.PassDoubleDash)
	parser.Name = Name
	parser.Usage = `[OPTIONS] [FILE]...

lsti extracts timing information from LS-DYNA message file(s) (e.g. messag, mes****)
File path accepts Unix style glob pattern (e.g. mes*, ./**/messag)

Example:
lsti messag
lsti ./**/mes* -o csv -f timigns.csv
lsti ./**/mes* --query "[].{path:File, elem:ElementProcessing}"
  `

	arguments, err := parser.Parse()
	if err != nil {
		fmt.Fprintln(cli.errStream, err)
		return ExitCodeError
	}

	// If arguments' length is zero or "-h, --help" flag is specified, show help and exit.
	if len(arguments) == 0 || opts.Misc.Help {
		parser.WriteHelp(os.Stdout)
		return ExitCodeOK
	}

	// Show version and exit.
	if opts.Misc.Version {
		fmt.Fprintf(cli.outStream, "%s version %s\n", Name, Version)
		return ExitCodeOK
	}

	// Expand glob pattern.
	var files []string
	for _, pattern := range arguments {
		matches, err := zglob.Glob(pattern)
		if err != nil {
			fmt.Fprintf(cli.errStream, "Invalid file path or glob pattern: %s", pattern)
		}
		files = append(files, matches...)
	}

	// If no files found, return error code and exit.
	if len(files) == 0 {
		fmt.Fprintf(cli.errStream, "No files found matching: %s", arguments)
		return ExitCodeError
	}

	// Parse files.
	schema, records, _ := cli.ParseMessageFiles(files)

	// Output parsed data in specified format.
	if err := cli.Write(schema, records); err != nil {
		fmt.Fprintln(cli.errStream, err)
		return ExitCodeError
	}

	return ExitCodeOK
}
