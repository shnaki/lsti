package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
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
	end := false
	count := 0
	var currentParent *Parent
	for scanner.Scan() {
		line := scanner.Text()

		// Search for header information.
		if !start {
			if strings.Contains(line, "Version : ") {
				record.Version = parseText([]rune(line), 18, 34)
				record.Date = parseText([]rune(line), 34, 55)
				continue
			}
			if strings.Contains(line, "Revision: ") {
				record.Revision, _ = parseInt([]rune(line), 18, 34)
				record.Time = parseText([]rune(line), 34, 55)
				continue
			}
			if strings.Contains(line, "Licensed to: ") {
				record.LicensedTo = parseText([]rune(line), 21, 55)
				continue
			}
			if strings.Contains(line, "Issued by  : ") {
				record.IssuedBy = parseText([]rune(line), 21, 55)
				continue
			}
			if strings.Contains(line, "Platform   : ") {
				record.Platform = parseText([]rune(line), 21, 55)
				continue
			}
			if strings.Contains(line, "OS Level   : ") {
				record.Os = parseText([]rune(line), 21, 55)
				continue
			}
			if strings.Contains(line, "Compiler   : ") {
				record.Compiler = parseText([]rune(line), 21, 55)
				continue
			}
			if strings.Contains(line, "Hostname   : ") {
				record.Hostname = parseText([]rune(line), 21, 55)
				continue
			}
			if strings.Contains(line, "Precision  : ") {
				record.Precision = parseText([]rune(line), 21, 55)
				continue
			}
			if strings.Contains(line, "SVN Version: ") {
				record.SvnVersion, _ = parseInt([]rune(line), 21, 55)
				continue
			}
			if strings.Contains(line, "Input file: ") {
				record.InputFile = parseText([]rune(line), 13, 84)
				continue
			}
		}

		// Search for timing information block.
		if strings.HasPrefix(line, " T i m i n g   i n f o r m a t i o n") {
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
			end = true
			continue
		}

		// Parse timing information.
		if start && !end {
			isParent := !strings.HasPrefix(line, "    ")
			runes := []rune(line)
			name := parseName(runes, 0, 25)
			cpuSec, _ := parseFloat(runes, 25, 36)
			cpuPercent, _ := parseFloat(runes, 36, 44)
			clockSec, _ := parseFloat(runes, 44, 59)
			clockPercent, _ := parseFloat(runes, 56, 67)
			if isParent {
				// Parent
				currentParent = record.AddParent(schema, name, cpuSec, cpuPercent, clockSec, clockPercent)
			} else {
				// Child
				currentParent.AddChild(schema, name, cpuSec, cpuPercent, clockSec, clockPercent)
			}
		}

		// Search for footer information.
		if end {
			if strings.HasPrefix(line, " N o r m a l    t e r m i n a t i o n") {
				record.NormalTermination = true
				continue
			}
			if strings.HasPrefix(line, " Elapsed time") {
				hours, _ := parseInt([]rune(line), 30, 33)
				minutes, _ := parseInt([]rune(line), 40, 43)
				seconds, _ := parseInt([]rune(line), 48, 52)
				d := (time.Duration(hours)*time.Hour +
					time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second)
				record.ElapsedTime = d.String()
				continue
			}
		}
	}
	return &record, nil
}

func parseName(runes []rune, start, end int) string {
	str := string(runes[start:end])
	return strings.TrimRight(strings.TrimRight(strings.Trim(str, " "), "."), " ")
}

func parseText(runes []rune, start, end int) string {
	str := string(runes[start:end])
	return strings.Trim(str, " ")
}

func parseInt(runes []rune, start, end int) (int64, error) {
	str := string(runes[start:end])
	str = strings.Trim(str, " ")
	return strconv.ParseInt(str, 10, 64)
}

func parseFloat(runes []rune, start, end int) (float64, error) {
	str := string(runes[start:end])
	str = strings.Trim(str, " ")
	return strconv.ParseFloat(str, 64)
}
