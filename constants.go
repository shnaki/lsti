package main

// Name and Version are showed in help message and version message.
const (
	Name    = "lsti"
	Version = "1.0.0"
)

// Exit codes are int values that represent an exit code for a particular error.
const (
	ExitCodeOK = iota
	ExitCodeError
)

// Option strings are string values that represent input string sets of limited values.
const (
	// (-d, --duration) option
	Human   = "human-readable"
	Seconds = "seconds"

	// (-f, --format) option
	Csv    = "csv"
	Html   = "html"
	Json   = "json"
	Simple = "simple"
	Table  = "table"
	Tsv    = "tsv"

	// (-t, --target) option
	CpuSec       = "cpusec"
	CpuPercent   = "pcpu"
	ClockSec     = "clocksec"
	ClockPercent = "pclock"
)
