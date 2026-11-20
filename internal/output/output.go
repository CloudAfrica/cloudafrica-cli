// Package output renders command results as a table, JSON or YAML.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"sigs.k8s.io/yaml"
)

// Format is an output format.
type Format string

const (
	Table Format = "table"
	JSON  Format = "json"
	YAML  Format = "yaml"
)

// Resolve picks the format: the flag when given, else table on a terminal
// and JSON otherwise, so a pipe gets something a program can read.
func Resolve(flag string, tty bool) (Format, error) {
	switch Format(strings.ToLower(flag)) {
	case "":
		if tty {
			return Table, nil
		}
		return JSON, nil
	case Table:
		return Table, nil
	case JSON:
		return JSON, nil
	case YAML:
		return YAML, nil
	}
	return "", fmt.Errorf("unknown output format %q: use table, json or yaml", flag)
}

// Result is what a command prints: the API value for json/yaml, its rows for
// a table, and the ids for --quiet.
type Result struct {
	Value   any
	Headers []string
	Rows    [][]string
	IDs     []string
}

// Printer writes results in one format.
type Printer struct {
	W      io.Writer
	Format Format
	Quiet  bool
}

// Print writes r.
func (p Printer) Print(r Result) error {
	if p.Quiet {
		for _, id := range r.IDs {
			if _, err := fmt.Fprintln(p.W, id); err != nil {
				return err
			}
		}
		return nil
	}
	switch p.Format {
	case JSON:
		enc := json.NewEncoder(p.W)
		enc.SetIndent("", "  ")
		return enc.Encode(r.Value)
	case YAML:
		data, err := yaml.Marshal(r.Value)
		if err != nil {
			return err
		}
		_, err = p.W.Write(data)
		return err
	default:
		tw := tabwriter.NewWriter(p.W, 0, 0, 2, ' ', 0)
		if len(r.Headers) > 0 {
			fmt.Fprintln(tw, strings.Join(r.Headers, "\t"))
		}
		for _, row := range r.Rows {
			fmt.Fprintln(tw, strings.Join(row, "\t"))
		}
		return tw.Flush()
	}
}

// S dereferences an optional string for a table cell.
func S(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

// N formats an optional number for a table cell.
func N[T ~int32 | ~int64 | ~int](v *T) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(*v)
}
