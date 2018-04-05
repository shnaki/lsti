package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/jmespath/go-jmespath"
	"github.com/olekukonko/tablewriter"
	"github.com/russross/blackfriday"
)

// Write results to stdout.
func (cli *CLI) Write(records []*Record) error {
	ds := cli.NormalizeRecords(records)

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

	// Format result string to specified format.
	str := ""
	switch opts.Out.Output {
	case Csv:
		str = cli.FormatSeparatedValues(data, ',', true)
	case Html:
		str = cli.FormatHtml(data)
	case Json:
		str = string(data) + "\n"
	case Table:
		str = cli.FormatTable(data)
	case Tsv:
		str = cli.FormatSeparatedValues(data, '	', false)
	}

	// Write to stdout.
	fmt.Fprint(cli.outStream, str)
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
func (cli *CLI) NormalizeRecords(records []*Record) []interface{} {
	dataType := opts.Out.Target
	var jsonSet []interface{}
	verbosity := len(opts.Out.Verbose)
	for _, record := range records {
		var jsonOut RecordData

		// Set properties.
		properties := make([]*JsonData, 0)
		properties = append(properties, &JsonData{Name: "file", Value: record.File})
		if verbosity >= 1 {
			if opts.Out.Duration == Human {
				properties = append(properties, &JsonData{Name: "elapsedTime", Value: formatSeconds(record.ElapsedTime)})
			} else {
				properties = append(properties, &JsonData{Name: "elapsedTime", Value: record.ElapsedTime})
			}
			properties = append(properties, &JsonData{Name: "version", Value: record.Version})
			properties = append(properties, &JsonData{Name: "svnVersion", Value: record.SvnVersion})
			properties = append(properties, &JsonData{Name: "platform", Value: record.Platform})
			properties = append(properties, &JsonData{Name: "compiler", Value: record.Compiler})
		}
		if verbosity >= 2 {
			properties = append(properties, &JsonData{Name: "NumCpus", Value: record.NumCpus})
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
				value := p.GetValue(dataType)
				if opts.Out.Duration == Human && (dataType == CpuSec || dataType == ClockSec) {
					timing.Value = formatSeconds(value)
				} else {
					timing.Value = value
				}
				timing.Details = make([]*JsonData, 0)
				pt = &timing
				timings = append(timings, &timing)
				return
			}
			if !opts.Out.Simple {
				if c, ok := d.(*Child); ok {
					js := JsonData{}
					js.Name = c.Name
					value := c.GetValue(dataType)
					if opts.Out.Duration == Human && (dataType == CpuSec || dataType == ClockSec) {
						js.Value = formatSeconds(value)
					} else {
						js.Value = value
					}
					pt.Details = append(pt.Details, &js)
					return
				}
			}
		})
		jsonOut.Timings = timings

		jsonSet = append(jsonSet, &jsonOut)
	}
	return jsonSet
}

func formatSeconds(seconds float64) string {
	d := time.Duration(seconds) * time.Second
	h := int(math.Floor(d.Hours()))
	m := int(math.Floor(d.Minutes())) - h*60
	s := int(math.Floor(d.Seconds())) - h*3600 - m*60
	str := fmt.Sprintf("%d:%02d:%02d", h, m, s)
	return str
}

// FormatSeparatedValues formats output data to CSV (with keys) or TSV (without keys) format.
func (cli *CLI) FormatSeparatedValues(data []byte, separator rune, withKeys bool) string {
	str := ""
	var ds []*RecordData
	json.Unmarshal(data, &ds)

	buf := new(bytes.Buffer)
	writer := csv.NewWriter(buf)
	writer.Comma = separator

	// Write keys.
	header := cli.GetHeader(ds)
	if withKeys {
		writer.Write(header.GetKeys())
	}

	// Write values.
	rows := cli.GetData(ds, header)
	for _, values := range rows {
		writer.Write(values)
	}

	writer.Flush()
	str = buf.String()
	return str
}

// FormatTable formats output data to ASCII table format.
func (cli *CLI) FormatTable(data []byte) string {
	str := ""
	var ds []*RecordData
	json.Unmarshal(data, &ds)

	buf := new(bytes.Buffer)

	// Set header.
	header := cli.GetHeader(ds)

	table := tablewriter.NewWriter(buf)
	table.SetHeader(header.GetKeys())
	table.SetAutoFormatHeaders(false)
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")

	// Set data.
	rows := cli.GetData(ds, header)
	table.AppendBulk(rows)

	table.Render()
	str = buf.String()
	return str
}

// Header stores keys with no duplicate.
type Header struct {
	PropertyKeys []string
	TimingKeys   []*TimingKey
}

// TimingKey stores keys with parent-child relationship.
type TimingKey struct {
	ParentKey string
	ChildKeys []string
}

// AddPropertyKey adds key of properties to header with no duplicate.
func (header *Header) AddPropertyKey(key string) {
	found := false
	for _, p := range header.PropertyKeys {
		if p == key {
			found = true
			break
		}
	}
	if !found {
		header.PropertyKeys = append(header.PropertyKeys, key)
	}
}

// AddParentKey adds key of parent to header with no duplicate.
func (header *Header) AddParentKey(key string) {
	found := false
	for _, p := range header.TimingKeys {
		if p.ParentKey == key {
			found = true
			break
		}
	}
	if !found {
		header.TimingKeys = append(header.TimingKeys, &TimingKey{ParentKey: key})
	}
}

// AddChildKey adds key of child to header with no duplicate.
func (header *Header) AddChildKey(parentKey, childKey string) {
	found := false
	var pt *TimingKey
	for _, p := range header.TimingKeys {
		if p.ParentKey == parentKey {
			pt = p
			for _, c := range p.ChildKeys {
				if c == childKey {
					found = true
					break
				}
			}
		}
	}
	if !found {
		strings := append(pt.ChildKeys, childKey)
		pt.ChildKeys = strings
	}
}

// GetHeader returns string array of keys.
func (header *Header) GetKeys() []string {
	var timingKeys []string
	for _, timingKey := range header.TimingKeys {
		timingKeys = append(timingKeys, timingKey.ParentKey)
		timingKeys = append(timingKeys, timingKey.ChildKeys...)
	}
	return append(header.PropertyKeys, timingKeys...)
}

// GetHeader returns header data for table.
func (cli *CLI) GetHeader(records []*RecordData) Header {
	var header Header

	// Add property keys.
	for _, record := range records {
		for _, property := range record.Properties {
			header.AddPropertyKey(property.Name)
		}
	}

	// Add timing keys.
	for _, record := range records {
		for _, timing := range record.Timings {
			header.AddParentKey(timing.Name)
			for _, detail := range timing.Details {
				header.AddChildKey(timing.Name, detail.Name)
			}
		}
	}
	return header
}

// GetData returns table data.
func (cli *CLI) GetData(records []*RecordData, header Header) [][]string {
	naWord := opts.Out.Miss
	var data [][]string

	for _, record := range records {
		// Get property data.
		var values []string
		for _, propertyKey := range header.PropertyKeys {
			for _, property := range record.Properties {
				if property.Name == propertyKey {
					values = append(values, fmt.Sprint(property.Value))
				}
			}
		}

		// Get timing data.
		for _, timingKey := range header.TimingKeys {
			// Get parent data.
			parentKey := timingKey.ParentKey
			parentFound := false
			for _, timing := range record.Timings {
				if timing.Name == parentKey {
					parentFound = true
					values = append(values, fmt.Sprint(timing.Value))
				}
			}
			if !parentFound {
				values = append(values, naWord)
			}

			// Get child data.
			for _, childKey := range timingKey.ChildKeys {
				childFound := false
				for _, timing := range record.Timings {
					if timing.Name == parentKey {
						for _, detail := range timing.Details {
							if detail.Name == childKey {
								childFound = true
								values = append(values, fmt.Sprint(detail.Value))
							}
						}
					}
				}
				if !childFound {
					values = append(values, naWord)
				}
			}
		}
		data = append(data, values)
	}
	return data
}

// FormatHtml formats output data to html table.
func (cli *CLI) FormatHtml(data []byte) string {
	var md = cli.FormatTable(data)
	html := blackfriday.MarkdownCommon([]byte(md))
	return string(html)
}
