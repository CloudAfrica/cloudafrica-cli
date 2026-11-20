package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"

	cloudafrica "github.com/CloudAfrica/cloudafrica-go"
)

func (a *App) apiCmd() *cobra.Command {
	var data string
	cmd := &cobra.Command{
		Use:   "api METHOD PATH",
		Short: "Make an authenticated API call",
		Long: "Send METHOD PATH (e.g. GET /v1/vms) to the API with the selected credential and print the response body. " +
			"--data sends a JSON body: a literal, @file, or @- for stdin. Writes get a fresh Idempotency-Key.",
		Args: exactArgs(2, "METHOD PATH"),
		RunE: func(cmd *cobra.Command, args []string) error {
			method := strings.ToUpper(args[0])
			path := args[1]
			if !strings.HasPrefix(path, "/") {
				return usageErr(fmt.Errorf("PATH must start with /, e.g. /v1/vms (got %q)", path))
			}
			body, err := a.readData(data)
			if err != nil {
				return usageErr(err)
			}
			return a.withSession(cmd.Context(), func(s *session) error {
				var rd io.Reader
				if body != nil {
					rd = bytes.NewReader(body)
				}
				req, err := http.NewRequestWithContext(cmd.Context(), method, s.endpoint+path, rd)
				if err != nil {
					return usageErr(err)
				}
				req.Header.Set("Accept", "application/json")
				if body != nil {
					req.Header.Set("Content-Type", "application/json")
				}
				resp, err := rawDo(s, req)
				if err != nil {
					return err
				}
				defer resp.Body.Close()
				out, err := io.ReadAll(resp.Body)
				if err != nil {
					return err
				}
				if err := cloudafrica.CheckResponse(resp, out); err != nil {
					// The problem document goes to stdout as well, for scripts.
					_, _ = a.Stdout.Write(out)
					return err
				}
				if loc := resp.Header.Get("Location"); loc != "" && resp.StatusCode == http.StatusAccepted && len(out) == 0 {
					fmt.Fprintf(a.Stdout, "{\"location\":%q}\n", loc)
					return nil
				}
				_, err = a.Stdout.Write(out)
				if err == nil && len(out) > 0 && out[len(out)-1] != '\n' {
					_, err = io.WriteString(a.Stdout, "\n")
				}
				return err
			})
		},
	}
	cmd.Flags().StringVarP(&data, "data", "d", "", "JSON request body: a literal, @file or @- (stdin)")
	return cmd
}

func (a *App) readData(d string) ([]byte, error) {
	switch {
	case d == "":
		return nil, nil
	case d == "@-":
		return io.ReadAll(a.Stdin)
	case strings.HasPrefix(d, "@"):
		b, err := os.ReadFile(d[1:])
		if err != nil {
			return nil, fmt.Errorf("reading --data: %w", err)
		}
		return b, nil
	}
	if strings.TrimSpace(d) == "" {
		return nil, errors.New("--data is empty")
	}
	return []byte(d), nil
}
