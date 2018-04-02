package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/jmespath/go-jmespath"
	"github.com/olekukonko/tablewriter"
	"io/ioutil"
	"os"
)

// Write outputs result to stdout and file.
func (cli *CLI) Write(schema *Schema, records []*Record) error {
	ds := cli.NormalizeRecords(schema, records)

	data, err := json.MarshalIndent(ds, "", "  ")
	if err != nil {
		return err
	}

	// If "-q, --query" is specified, apply JMESPath to json.
	expression := opts.Out.Query
	if expression != "" {
		data, err = cli.Query(data, expression)
		if err != nil {
			return err
		}
	}

	// Format output string to specified format.
	str := ""
	if len(records) != 1 {
		schema.Formatter = append([]string{"File"}, schema.Formatter...)
	}
	switch opts.Out.Output {
	case "json":
		str = string(data) + "\n"
	case "csv":
		str = cli.FormatSeparatedValues(data, schema, ',', true)
	case "tsv":
		str = cli.FormatSeparatedValues(data, schema, '	', false)
	case "table":
		str = cli.FormatTable(data, schema)
	}

	// Print to stdout.
	if !opts.Out.Quiet {
		fmt.Fprint(cli.outStream, str)
	}

	// Write to file.
	filename := opts.Out.File
	if filename != "" {
		content := []byte(str)
		ioutil.WriteFile(filename, content, os.ModePerm)
	}

	return nil
}

// Query applies JMESPath to json.
func (cli *CLI) Query(data []byte, expression string) ([]byte, error) {
	var d interface{}
	json.Unmarshal(data, &d)
	jp, err := jmespath.Compile(expression)
	if err != nil {
		return nil, err
	}
	result, err := jp.Search(d)
	if err != nil {
		return nil, err
	}

	toJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, err
	}
	data = toJSON
	return data, nil
}

// A JsonOut stores data for json.MarshalIndent.
type JsonOut struct {
	File    string    `json:"file"`
	Timings []*Timing `json:"timings"`
}

// A Timing represents timing information struct that has parent-child relationship for json MarshalIndent.
type Timing struct {
	JsonData
	Details []*JsonData `json:"details"`
}

// A JsonData represents general data struct for json MarshalIndent.
type JsonData struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

// NormalizeRecords normalizes records for json output.
func (cli *CLI) NormalizeRecords(schema *Schema, records []*Record) []*JsonOut {
	dataType := opts.Out.Target
	var jsonSet []*JsonOut
	for _, record := range records {
		jsonOut := JsonOut{}
		jsonOut.File = record.File
		jsonOut.Timings = make([]*Timing, 0)
		var pt *Timing
		record.ForEachData(func(d interface{}, _ int) {
			if p, ok := d.(*Parent); ok {
				timing := Timing{}
				timing.Name = p.Name
				timing.Value = p.GetValue(dataType)
				timing.Details = make([]*JsonData, 0)
				pt = &timing
				jsonOut.Timings = append(jsonOut.Timings, &timing)
				return
			}
			if c, ok := d.(*Child); ok {
				js := JsonData{}
				js.Name = c.Name
				js.Value = c.GetValue(dataType)
				pt.Details = append(pt.Details, &js)
				return
			}
		})
		jsonSet = append(jsonSet, &jsonOut)
	}
	return jsonSet
}

// FormatSeparatedValues formats output data to CSV (with keys) or TSV (without keys) format.
func (cli *CLI) FormatSeparatedValues(data []byte, schema *Schema, separator rune, withKeys bool) string {
	str := ""
	var ds []map[string]interface{}
	json.Unmarshal(data, &ds)

	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)
	writer.Comma = separator

	// Write keys.
	if withKeys {
		keys := cli.GetKeys(schema, ds)
		writer.Write(keys)
	}

	// Write values.
	rows := cli.GetData(ds, schema)
	for _, values := range rows {
		writer.Write(values)
	}

	writer.Flush()
	str = buf.String()
	return str
}

// FormatTable formats output data to ASCII table format.
func (cli *CLI) FormatTable(data []byte, schema *Schema) string {
	str := ""
	var ds []map[string]interface{}
	json.Unmarshal(data, &ds)

	buf := new(bytes.Buffer)

	// Set header.
	keys := cli.GetKeys(schema, ds)

	table := tablewriter.NewWriter(buf)
	table.SetHeader(keys)
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")

	// Set data.
	rows := cli.GetData(ds, schema)
	table.AppendBulk(rows)

	table.Render()
	str = buf.String()
	return str
}

// GetKeys returns a slice of keys.
func (cli *CLI) GetKeys(schema *Schema, ds []map[string]interface{}) []string {
	//var keys []string

	schema.Formatter = make([]string, 0)
	forEachKey(ds, func(key string, value interface{}, nodeType int) {
		switch nodeType {
		case FIELD:
			schema.AddPath(schema.normalizePath(key))
		case PARENT:
			schema.AddPath(schema.normalizePath(key))
		case CHILD:
			schema.AddPath(schema.normalizePath(key))
		}
	})
	return schema.Formatter
}

const (
	FIELD = iota
	PARENT
	CHILD
)

func forEachKey(ds []map[string]interface{}, cb func(key string, value interface{}, nodeType int)) {
	for _, d := range ds {
		for key, value := range d {
			switch value.(type) {
			case string:
				cb(key, value, FIELD)
			case []interface{}:
				if timings, ok := value.([]interface{}); ok {
					for _, timing := range timings {
						if t, ok := timing.(map[string]interface{}); ok {
							pn := fmt.Sprint(t["name"])
							pv := fmt.Sprint(t["value"])
							cb(pn, pv, PARENT)
							d := t["details"]
							if details, ok := d.([]interface{}); ok {
								for _, detail := range details {
									if t, ok := detail.(map[string]interface{}); ok {
										cn := fmt.Sprint(t["name"])
										cv := fmt.Sprint(t["value"])
										cb(cn, cv, CHILD)
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

// GetData returns table data.
func (cli *CLI) GetData(ds interface{}, schema *Schema) [][]string {
	var data [][]string
	if s, ok := ds.([]interface{}); ok {
		for _, d := range s {
			var values []string
			if m, ok := d.(map[string]interface{}); ok {
				for _, key := range schema.Formatter {
					val := m[key]
					if val == nil {
						val = 0.0
					}
					strValue := fmt.Sprint(val)
					values = append(values, strValue)
				}
			}
			data = append(data, values)
		}
	}
	return data
}
