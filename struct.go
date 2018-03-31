package main

// A Schema stores table format.
type Schema struct {
	Formatter []string
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
	Children []*Child `json:"children"`
}

// AddChild adds child data, and register path to Schema.
func (parent *Parent) AddChild(schema *Schema, name string, cpuSec, cpuPercent, clockSec, clockPercent float64) *Child {
	dataPath := parent.Path + "#" + name
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
	File    string
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
	parent := Parent{}
	parent.Name = name
	parent.Path = name
	parent.CpuSec = cpuSec
	parent.CpuPercent = cpuPercent
	parent.ClockSec = clockSec
	parent.ClockPercent = clockPercent
	record.Parents = append(record.Parents, &parent)
	schema.AddPath(parent.Path)
	return &parent
}
