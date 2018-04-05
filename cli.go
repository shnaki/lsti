package main

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
	Csv   = "csv"
	Json  = "json"
	Table = "table"
	Tsv   = "tsv"

	// (-t, --target) option
	CpuSec       = "cpusec"
	CpuPercent   = "pcpu"
	ClockSec     = "clocksec"
	ClockPercent = "pclock"
)
