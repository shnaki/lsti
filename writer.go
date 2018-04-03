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

// A RecordData represents record for json MarshalIndent.
type RecordData struct {
	Properties []*JsonData   `json:"properties"`
	Timings    []*TimingData `json:"details"`
}

// A TimingData represents timing information struct that has parent-child relationship for json MarshalIndent.
type TimingData struct {
	JsonData
	Details []*JsonData `json:"details"`
}

// A JsonData represents general data struct for json MarshalIndent.
type JsonData struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

// NormalizeRecords normalizes records for json output.
func (cli *CLI) NormalizeRecords(schema *Schema, records []*Record) []interface{} {
	dataType := opts.Out.Target
	var jsonSet []interface{}
	verbosity := len(opts.Out.Verbose)
	for _, record := range records {
		var jsonOut RecordData

		// Set properties.
		properties := make([]*JsonData, 0)
		properties = append(properties, &JsonData{Name: "file", Value: record.File})
		if verbosity >= 1 {
			properties = append(properties, &JsonData{Name: "elapsedTime", Value: record.ElapsedTime})
			properties = append(properties, &JsonData{Name: "version", Value: record.Version})
			properties = append(properties, &JsonData{Name: "svnVersion", Value: record.SvnVersion})
			properties = append(properties, &JsonData{Name: "platform", Value: record.Platform})
			properties = append(properties, &JsonData{Name: "compiler", Value: record.Compiler})
		}
		if verbosity >= 2 {
			properties = append(properties, &JsonData{Name: "os", Value: record.Os})
			properties = append(properties, &JsonData{Name: "inputFile", Value: record.InputFile})
			properties = append(properties, &JsonData{Name: "hostname", Value: record.Hostname})
		}
		if verbosity >= 3 {
			properties = append(properties, &JsonData{Name: "revision", Value: record.Revision})
			properties = append(properties, &JsonData{Name: "precision", Value: record.Precision})
			properties = append(properties, &JsonData{Name: "licensedTo", Value: record.LicensedTo})
			properties = append(properties, &JsonData{Name: "issuedBy", Value: record.IssuedBy})
			properties = append(properties, &JsonData{Name: "normalTermination", Value: record.NormalTermination})
		}
		jsonOut.Properties = properties

		// Set timings.
		timings := make([]*TimingData, 0)
		var pt *TimingData
		record.ForEachData(func(d interface{}, _ int) {
			if p, ok := d.(*Parent); ok {
				timing := TimingData{}
				timing.Name = p.Name
				timing.Value = p.GetValue(dataType)
				timing.Details = make([]*JsonData, 0)
				pt = &timing
				timings = append(timings, &timing)
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
		jsonOut.Timings = timings

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
