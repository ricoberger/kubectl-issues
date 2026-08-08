package writer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

func WriteResults(out io.Writer, headers []string, matrix [][]string, noHeader bool) {
	w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	defer w.Flush()

	if len(matrix) == 0 {
		fmt.Fprintln(w, "No resources found")
	} else {
		if !noHeader {
			fmt.Fprintln(w, strings.Join(headers, "\t"))
		}

		for _, row := range matrix {
			fmt.Fprintln(w, strings.Join(row, "\t"))
		}
	}
}

// jsonRow marshals a single table row as a JSON object which uses the table
// headers as keys, preserving the column order of the table.
type jsonRow struct {
	headers []string
	values  []string
}

func (r jsonRow) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')

	for i, header := range r.headers {
		if i > 0 {
			buf.WriteByte(',')
		}

		key, err := json.Marshal(header)
		if err != nil {
			return nil, err
		}
		buf.Write(key)
		buf.WriteByte(':')

		value, err := json.Marshal(r.values[i])
		if err != nil {
			return nil, err
		}
		buf.Write(value)
	}

	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// WriteJSON writes the results as a JSON array of objects, so they can be
// consumed by other applications. The keys of each object are the table
// headers in column order, the values are the formatted strings of the table,
// so that a consumer can reprint the identical table. An empty matrix is
// written as an empty array.
func WriteJSON(out io.Writer, headers []string, matrix [][]string) error {
	rows := make([]jsonRow, 0, len(matrix))
	for _, row := range matrix {
		rows = append(rows, jsonRow{headers: headers, values: row})
	}

	result, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}

	_, err = fmt.Fprintf(out, "%s\n", result)
	return err
}
