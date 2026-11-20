package cli

import (
	"context"
	"errors"
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

func (a *App) sshKeyCmd() *cobra.Command {
	cmd := group("ssh-key", "List your SSH keys")
	cmd.AddCommand(listCmd(a, "List your SSH keys (ids for vm create --ssh-key)",
		[]string{"ID", "NAME", "FINGERPRINT"},
		func(k cloudafrica.PublicSshKeyDto) []string {
			return []string{output.S(k.Id), output.S(k.Name), output.S(k.Fingerprint)}
		},
		func(k cloudafrica.PublicSshKeyDto) string { return output.S(k.Id) },
		func(ctx context.Context, c *cloudafrica.ClientWithResponses) ([]cloudafrica.PublicSshKeyDto, error) {
			return listAll(func(cursor *string) (*[]cloudafrica.PublicSshKeyDto, *string, error) {
				limit := pageLimit
				r, err := c.ListSshKeysWithResponse(ctx, &cloudafrica.ListSshKeysParams{Cursor: cursor, Limit: &limit})
				if err != nil {
					return nil, nil, err
				}
				if err := cloudafrica.CheckResponse(r.HTTPResponse, r.Body); err != nil {
					return nil, nil, err
				}
				if r.JSON200 == nil {
					return nil, nil, errors.New("empty SSH key list response")
				}
				return r.JSON200.Data, r.JSON200.NextCursor, nil
			})
		}))
	return cmd
}

func (a *App) vpcCmd() *cobra.Command {
	cmd := group("vpc", "List VPCs")
	cmd.AddCommand(listCmd(a, "List VPCs (ids for vm create --vpc)",
		[]string{"ID", "NAME", "STATUS", "DESCRIPTION"},
		func(v cloudafrica.PublicVpcDto) []string {
			return []string{output.S(v.Id), output.S(v.Name), output.S(v.Status), output.S(v.Description)}
		},
		func(v cloudafrica.PublicVpcDto) string { return output.S(v.Id) },
		func(ctx context.Context, c *cloudafrica.ClientWithResponses) ([]cloudafrica.PublicVpcDto, error) {
			return listAll(func(cursor *string) (*[]cloudafrica.PublicVpcDto, *string, error) {
				limit := pageLimit
				r, err := c.ListVpcsWithResponse(ctx, &cloudafrica.ListVpcsParams{Cursor: cursor, Limit: &limit})
				if err != nil {
					return nil, nil, err
				}
				if err := cloudafrica.CheckResponse(r.HTTPResponse, r.Body); err != nil {
					return nil, nil, err
				}
				if r.JSON200 == nil {
					return nil, nil, errors.New("empty VPC list response")
				}
				return r.JSON200.Data, r.JSON200.NextCursor, nil
			})
		}))
	return cmd
}

func (a *App) datacenterCmd() *cobra.Command {
	cmd := group("datacenter", "List datacenters")
	cmd.AddCommand(listCmd(a, "List the datacenters this account can place into (slugs for vm create --datacenter)",
		[]string{"SLUG", "LABEL"},
		func(d cloudafrica.PublicPlacementDatacenterDto) []string {
			return []string{output.S(d.Slug), output.S(d.Label)}
		},
		func(d cloudafrica.PublicPlacementDatacenterDto) string { return output.S(d.Slug) },
		func(ctx context.Context, c *cloudafrica.ClientWithResponses) ([]cloudafrica.PublicPlacementDatacenterDto, error) {
			// One page: the route takes no cursor and answers every datacenter.
			r, err := c.ListPlacementGroupDatacentersWithResponse(ctx)
			if err != nil {
				return nil, err
			}
			if err := cloudafrica.CheckResponse(r.HTTPResponse, r.Body); err != nil {
				return nil, err
			}
			if r.JSON200 == nil {
				return nil, errors.New("empty datacenter list response")
			}
			if r.JSON200.Data == nil {
				return []cloudafrica.PublicPlacementDatacenterDto{}, nil
			}
			return *r.JSON200.Data, nil
		}))
	return cmd
}
