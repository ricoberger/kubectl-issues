package writer

import (
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
