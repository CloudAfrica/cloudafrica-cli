package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	cloudafrica "github.com/CloudAfrica/cloudafrica-go"

	"github.com/CloudAfrica/cloudafrica-cli/internal/output"
)

const pageLimit = 200

// listAll follows nextCursor from the first page to the last. page returns
// one page's rows and its nextCursor; a cursor seen before is an error, not
// an endless loop.
func listAll[T any](page func(cursor *string) (data *[]T, next *string, err error)) ([]T, error) {
	out := []T{}
	var cursor *string
	seen := map[string]bool{}
	for {
		data, next, err := page(cursor)
		if err != nil {
			return nil, err
		}
		if data != nil {
			out = append(out, *data...)
		}
		if next == nil || *next == "" {
			return out, nil
		}
		if seen[*next] {
			return nil, fmt.Errorf("the server repeated the page cursor %q", *next)
		}
		seen[*next] = true
		cursor = next
	}
}

// listCmd is a `list` subcommand that prints every row fetch returns.
func listCmd[T any](a *App, short string, headers []string, row func(T) []string, id func(T) string,
	fetch func(ctx context.Context, c *cloudafrica.ClientWithResponses) ([]T, error)) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: short,
		Args:  noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			pr, err := a.printer()
			if err != nil {
				return err
			}
			return a.withSession(cmd.Context(), func(s *session) error {
				rows, err := fetch(cmd.Context(), s.client.API)
				if err != nil {
					return err
				}
				res := output.Result{Value: rows, Headers: headers}
				for _, r := range rows {
					res.Rows = append(res.Rows, row(r))
					res.IDs = append(res.IDs, id(r))
				}
				return pr.Print(res)
			})
		},
	}
}
