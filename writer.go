package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/jmespath/go-jmespath"
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
		schema.Formatter = append([]string{"file"}, schema.Formatter...)
	}
	switch opts.Out.Output {
	case "json":
		str = string(data)
	case "csv":
		str = cli.FormatSeparatedValues(data, schema, ',', true)
	case "tsv":
		str = cli.FormatSeparatedValues(data, schema, '	', false)
	}

	// Print to stdout.
	fmt.Fprintln(cli.outStream, str)

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
	result, err := jmespath.Search(expression, d)
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

// NormalizeRecords normalizes records to array of map.
func (cli *CLI) NormalizeRecords(schema *Schema, records []*Record) []map[string]interface{} {
	dataType := opts.Out.Data
	var ds []map[string]interface{}
	for _, record := range records {
		data := make(map[string]interface{})
		data["file"] = record.File
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
		var keys []string
		for _, format := range schema.Formatter {
			elements := strings.Split(format, "#")
			key := ""
			if len(elements) == 1 {
				key = elements[0]
			} else {
				key = elements[1]
			}
			keys = append(keys, key)
		}
		writer.Write(keys)
	}

	// Write values.
	if s, ok := ds.([]interface{}); ok {
		for _, data := range s {
			var values []string
			if m, ok := data.(map[string]interface{}); ok {
				for _, key := range schema.Formatter {
					val := m[key]
					if val == nil {
						val = 0.0
					}
					strValue := fmt.Sprint(val)
					values = append(values, strValue)
				}
			}
			writer.Write(values)
		}
	}
	writer.Flush()
	str = buf.String()
	return str
}
