package main

import (
	"fmt"
	"io"
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/mattn/go-zglob"
)

var opts struct {
	Misc Misc   `group:"Miscellaneous"`
	Out  Output `group:"Output control"`
}

type Misc struct {
	Help    bool `short:"h" long:"help" description:"Show this help message and exit"`
	Version bool `short:"V" long:"version" description:"Show version information and exit"`
}

type Output struct {
	Abs      bool   `short:"a" long:"absolute" description:"Use absolute path for \"file\" property"`
	Duration string `short:"d" long:"duration" description:"Duration format\nhuman-readable means [h]:mm:ss, and rounds down floating point values" choice:"human-readable" choice:"seconds" default:"human-readable"`
	Miss     string `short:"m" long:"missing" description:"Replace missing values with specified string" default:"n/a"`
	Output   string `short:"o" long:"output" description:"Output format\n(default: simple for single file, table for multiple files)" choice:"csv" choice:"html" choice:"json" choice:"simple" choice:"table" choice:"tsv"`
	Query    string `short:"q" long:"query" description:"JMESPath query string\nSee http://jmespath.org/ for more information and examples"`
	Relative string `short:"r" long:"relative" description:"Use relative path for \"file\" property (relative to specified path)\nIf \"-a, --absolute\" option is specified, this option will be ignored"`
	Simple   bool   `short:"s" long:"simple" description:"Suppress detail timing information (e.g. Solids, Shells)"`
	Target   string `short:"t" long:"target" description:"Target value used for statistics" choice:"cpusec" choice:"pcpu" choice:"clocksec" choice:"pclock" default:"clocksec"`
	Verbose  []bool `short:"v" long:"verbose" description:"Output verbose information, this option can be specified multiple times\n-v:   + Output LS-DYNA module information and elapsed time\n-vv:  + Output execution environment\n-vvv: + Output more information"`
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

lsti extracts timing information from LS-DYNA message file(s) (e.g. messag, mes****),
and display results in the specified format
File path accepts Unix style glob pattern (e.g. mes*, ./**/messag)

Example:
  lsti mes0000
  lsti ./**/mes* -o csv > timigns.csv
  lsti ./**/mes* -o table > timigns.md
  lsti ./**/messag -vvv --query "[].{properties:properties[?name=='file' || name=='elapsedTime']}"
  `

	arguments, err := parser.Parse()
	if err != nil {
		fmt.Fprintln(cli.errStream, err)
		return ExitCodeError
	}

	// If arguments' length is zero, show help and exit with error.
	if len(arguments) == 0 {
		parser.WriteHelp(os.Stdout)
		return ExitCodeError
	}

	// If "-h, --help" flag is specified, show help and exit.
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
			fmt.Fprintf(cli.errStream, "Invalid file path or glob pattern: %s\n", pattern)
		}
		files = append(files, matches...)
	}

	// If no files found, return error code and exit.
	if len(files) == 0 {
		fmt.Fprintf(cli.errStream, "No files found matching: %s\n", arguments)
		return ExitCodeError
	}

	// Parse files.
	records, _ := cli.ParseMessageFiles(files)

	// Output parsed data in specified format.
	if err := cli.Write(records); err != nil {
		fmt.Fprintln(cli.errStream, err)
		return ExitCodeError
	}

	return ExitCodeOK
}
