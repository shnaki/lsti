package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ParseMessageFiles parses LS-DYNA message files (e.g. messag, mes****) and return Records.
func (cli *CLI) ParseMessageFiles(files []string) (*Schema, []*Record, error) {
	sort.Strings(files)
	var records []*Record
	schema := Schema{}
	for _, file := range files {
		record, err := cli.ParseMessageFile(&schema, file)
		if err != nil {
			fmt.Fprintln(cli.errStream, err)
		}
		records = append(records, record)
	}
	return &schema, records, nil
}

// ParseMessageFile parses LS-DYNA message file (e.g. messag, mes****) and return Record.
func (cli *CLI) ParseMessageFile(schema *Schema, file string) (*Record, error) {
	fp, err := os.Open(filepath.FromSlash(file))
	if err != nil {
		return nil, err
	}
	defer fp.Close()

	record := Record{File: file}
	scanner := bufio.NewScanner(fp)
	start := false
	count := 0
	var currentParent *Parent
	for scanner.Scan() {
		line := scanner.Text()

		// Search for timing information block.
		if strings.Contains(line, "T i m i n g   i n f o r m a t i o n") {
			start = true
			continue
		}
		if !start {
			continue
		}

		// Skip 2 header lines.
		count++
		if count <= 2 {
			continue
		}

		// If timing information block ends, stop reading.
		if strings.Contains(line, "-----------------------") {
			break
		}

		// Parse timing information.
		// TODO(tenchanome) Implement error handling.
		isParent := !strings.HasPrefix(line, "    ")
		runes := []rune(line)
		name := parseName(runes, 0, 25)
		cpuSec, err1 := parseFloat(runes, 25, 36)
		cpuPercent, err2 := parseFloat(runes, 36, 44)
		clockSec, err3 := parseFloat(runes, 44, 59)
		clockPercent, err4 := parseFloat(runes, 56, 67)
		_, _, _, _ = err1, err2, err3, err4
		if isParent {
			// Parent
			currentParent = record.AddParent(schema, name, cpuSec, cpuPercent, clockSec, clockPercent)
		} else {
			// Child
			currentParent.AddChild(schema, name, cpuSec, cpuPercent, clockSec, clockPercent)
		}
	}
	return &record, nil
}

func parseName(runes []rune, start, end int) string {
	str := string(runes[start:end])
	return strings.TrimRight(strings.TrimRight(strings.Trim(str, " "), "."), " ")
}

func parseFloat(runes []rune, start, end int) (float64, error) {
	str := string(runes[start:end])
	str = strings.Trim(str, " ")
	return strconv.ParseFloat(str, 64)
}
