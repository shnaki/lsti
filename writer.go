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
	"strings"
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

// NormalizeRecords normalizes records for json.
func (cli *CLI) NormalizeRecords(schema *Schema, records []*Record) []map[string]interface{} {
	dataType := opts.Out.Target
	var ds []map[string]interface{}
	for _, record := range records {
		data := make(map[string]interface{})
		data["File"] = record.File
		record.ForEachData(func(d interface{}, _ int) {
			if p, ok := d.(*Parent); ok {
				data[p.Path] = p.GetValue(dataType)
				return
			}
			if c, ok := d.(*Child); ok {
				data[c.Path] = c.GetValue(dataType)
				return
			}
		})
		ds = append(ds, data)
	}
	return ds
}

// FormatSeparatedValues formats output data to CSV (with keys) or TSV (without keys) format.
func (cli *CLI) FormatSeparatedValues(data []byte, schema *Schema, separator rune, withKeys bool) string {
	str := ""
	var ds interface{}
	json.Unmarshal(data, &ds)

	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)
	writer.Comma = separator

	// Write keys.
	if withKeys {
		keys := cli.GetKeys(schema)
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
	var ds interface{}
	json.Unmarshal(data, &ds)

	buf := new(bytes.Buffer)

	// Set header.
	keys := cli.GetKeys(schema)

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
func (cli *CLI) GetKeys(schema *Schema) []string {
	var keys []string
	for _, format := range schema.Formatter {
		originalFormat := schema.Names[format]
		if originalFormat != "" {
			format = originalFormat
		}
		elements := strings.Split(format, "/")
		key := ""
		if len(elements) == 1 {
			key = elements[0]
		} else {
			key = elements[1]
		}
		keys = append(keys, key)
	}
	return keys
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
