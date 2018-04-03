package main

import (
	"path"
	"strings"
	"unicode"
)

// A Schema stores table format.
type Schema struct {
	Formatter []string
	Names     map[string]string
}

// AddPath adds path to table.
func (schema *Schema) AddPath(path string) {
	found := false
	for _, p := range schema.Formatter {
		if p == path {
			found = true
			break
		}
	}
	if !found {
		schema.Formatter = append(schema.Formatter, path)
	}
}

func (schema *Schema) normalizePath(path string) string {
	elements := strings.Split(path, "/")
	for i, element := range elements {
		parts := strings.Fields(element)
		for i, part := range parts {
			runes := []rune(part)
			runes[0] = unicode.ToUpper(runes[0])
			parts[i] = string(runes)
		}
		str := strings.Join(parts, "")
		str = strings.Replace(str, ".", "", -1)
		elements[i] = str
	}
	normalizedPath := strings.Join(elements, "/")
	if schema.Names == nil {
		schema.Names = make(map[string]string, 20)
	}
	schema.Names[normalizedPath] = path
	return normalizedPath
}

// A Data represents the timing information parsed from LS-DYNA message file.
type Data struct {
	Name, Path                                 string
	CpuSec, CpuPercent, ClockSec, ClockPercent float64
}

// GetValue returns value used for aggregation.
func (data *Data) GetValue(dataType string) float64 {
	switch dataType {
	case "cpusec":
		return data.CpuSec
	case "pcpu":
		return data.CpuPercent
	case "clocksec":
		return data.ClockSec
	case "pclock":
		return data.ClockPercent
	}
	return 0.0
}

// A Child represents the child information (e.g. Solids, Shells).
type Child struct {
	Data
}

// Parent represents the parent information (e.g. Keyword Processing).
type Parent struct {
	Data
	Children []*Child
}

// AddChild adds child data, and register path to Schema.
func (parent *Parent) AddChild(schema *Schema, name string, cpuSec, cpuPercent, clockSec, clockPercent float64) *Child {
	dataPath := path.Join(parent.Path, name)
	dataPath = schema.normalizePath(dataPath)
	child := Child{}
	child.Name = name
	child.Path = dataPath
	child.CpuSec = cpuSec
	child.CpuPercent = cpuPercent
	child.ClockSec = clockSec
	child.ClockPercent = clockPercent
	parent.Children = append(parent.Children, &child)
	schema.AddPath(child.Path)
	return &child
}

// GetNumChildren returns the number of Children in this parent data.
func (parent *Parent) GetNumChildren() int {
	return len(parent.Children)
}

// ForEachChildren returns the number of Children in this parent data.
func (parent *Parent) ForEachChildren(cb func(*Child, int)) {
	for i, child := range parent.Children {
		cb(child, i)
	}
}

// Record represents the data set parsed from a LS-DYNA message file.
type Record struct {
	File string

	Version                                     string
	Revision                                    int64
	Date, Time                                  string
	LicensedTo, IssuedBy                        string
	Platform, Os, Compiler, Hostname, Precision string
	SvnVersion                                  int64

	InputFile string

	NormalTermination bool
	ElapsedTime       string

	Parents []*Parent
}

// GetNumParents returns the number of parents in this record.
func (record *Record) GetNumParents() int {
	return len(record.Parents)
}

// GetNumChildren returns the number of Children in this record.
func (record *Record) GetNumChildren() int {
	numChildren := 0
	record.ForEachParent(func(parent *Parent, i int) {
		numChildren += parent.GetNumChildren()
	})
	return numChildren
}

// GetNumData returns the number of data (num parents + num Children) in this record.
func (record *Record) GetNumData() int {
	numData := 0
	record.ForEachParent(func(parent *Parent, i int) {
		numData++
		numData += parent.GetNumChildren()
	})
	return numData
}

// ForEachParent executes callback function for each parent.
func (record *Record) ForEachParent(cb func(*Parent, int)) {
	for i, parent := range record.Parents {
		cb(parent, i)
	}
}

// ForEachData executes callback function for each data.
func (record *Record) ForEachData(cb func(interface{}, int)) {
	count := 0
	record.ForEachParent(func(parent *Parent, _ int) {
		cb(parent, count)
		count++
		parent.ForEachChildren(func(child *Child, _ int) {
			cb(child, count)
			count++
		})
	})
}

// ForEachChild executes callback function for each child.
func (record *Record) ForEachChild(cb func(interface{}, int)) {
	count := 0
	record.ForEachParent(func(parent *Parent, _ int) {
		cb(parent, count)
		count++
		parent.ForEachChildren(func(child *Child, _ int) {
			cb(child, count)
			count++
		})
	})
}

// AddParent adds parent data, and register path to Schema.
func (record *Record) AddParent(schema *Schema, name string, cpuSec, cpuPercent, clockSec, clockPercent float64) *Parent {
	dataPath := schema.normalizePath(name)
	parent := Parent{}
	parent.Name = name
	parent.Path = dataPath
	parent.CpuSec = cpuSec
	parent.CpuPercent = cpuPercent
	parent.ClockSec = clockSec
	parent.ClockPercent = clockPercent
	record.Parents = append(record.Parents, &parent)
	schema.AddPath(parent.Path)
	return &parent
}
