package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"

	cloudafrica "github.com/CloudAfrica/cloudafrica-go"

	"github.com/CloudAfrica/cloudafrica-cli/internal/output"
)

const defaultWaitTimeout = 10 * time.Minute

// waitFlags are --no-wait and --timeout on an async command, which waits by default.
type waitFlags struct {
	noWait  bool
	timeout time.Duration
	// cancelled makes a cancelled operation the outcome waited for (`operation cancel`).
	cancelled bool
}

func (w *waitFlags) bind(cmd *cobra.Command, withNoWait bool) {
	if withNoWait {
		cmd.Flags().BoolVar(&w.noWait, "no-wait", false, "print the operation and return without waiting for it")
	}
	cmd.Flags().DurationVar(&w.timeout, "timeout", defaultWaitTimeout, "how long to wait for the operation")
}

func (a *App) operationCmd() *cobra.Command {
	cmd := group("operation", "Inspect and cancel asynchronous operations")
	cmd.Aliases = []string{"op"}
	var wf waitFlags
	wait := &cobra.Command{
		Use:   "wait OPERATION_ID",
		Short: "Wait for an operation to finish",
		Args:  exactArgs(1, "OPERATION_ID"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.withSession(cmd.Context(), func(s *session) error {
				return a.waitAndPrint(cmd.Context(), s, args[0], wf)
			})
		},
	}
	wf.bind(wait, false)
	cmd.AddCommand(wait)
	return cmd
}

// rawDo sends req through the SDK client's transport (auth, retries, origin check).
func rawDo(s *session, req *http.Request) (*http.Response, error) {
	g, ok := s.client.API.ClientInterface.(*cloudafrica.Gen)
	if !ok {
		return nil, errors.New("internal: the SDK client has no raw transport")
	}
	return g.Client.Do(req)
}

// getOperation reads an operation. Ids contain "/" and are joined verbatim,
// which the generated GetOperation (it escapes the id) would break.
func getOperation(ctx context.Context, s *session, id string) (*cloudafrica.Operation, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.endpoint+cloudafrica.OperationPath(strings.TrimLeft(id, "/")), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := rawDo(s, req)
	if err != nil {
		return nil, err
	}
	var op cloudafrica.Operation
	if err := decodeJSON(resp, &op); err != nil {
		return nil, err
	}
	return &op, nil
}

func operationRow(op *cloudafrica.Operation) []string {
	target := ""
	if op.Target != nil {
		target = output.S(op.Target.Id)
	}
	created := ""
	if op.CreatedResource != nil {
		created = output.S(op.CreatedResource.Id)
	}
	msg := output.S(op.ProgressMessage)
	if op.Error != nil {
		msg = output.S(op.Error.Message)
	}
	return []string{output.S(op.Id), output.S(op.Kind), output.S(op.State), target, created, msg}
}

var operationHeaders = []string{"ID", "KIND", "STATE", "TARGET", "CREATED", "MESSAGE"}

func (a *App) printOperation(op *cloudafrica.Operation) error {
	pr, err := a.printer()
	if err != nil {
		return err
	}
	id := output.S(op.Id)
	if op.CreatedResource != nil && output.S(op.CreatedResource.Id) != "" && output.S(op.State) == "complete" {
		id = output.S(op.CreatedResource.Id)
	}
	return pr.Print(output.Result{Value: op, Headers: operationHeaders, Rows: [][]string{operationRow(op)}, IDs: []string{id}})
}

// waitAndPrint polls an operation until it finishes or timeout passes,
// writing its progress to stderr on a terminal. A failed or cancelled
// operation prints and returns its error; running out of time returns a
// deadline error (exit 3).
func (a *App) waitAndPrint(ctx context.Context, s *session, id string, wf waitFlags) error {
	timeout := wf.timeout
	if timeout <= 0 {
		timeout = defaultWaitTimeout
	}
	wctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	op, err := a.pollOperation(wctx, s, id)
	if err != nil {
		var oe *cloudafrica.OperationError
		if errors.As(err, &oe) && op != nil {
			if perr := a.printOperation(op); perr != nil {
				return perr
			}
			if wf.cancelled && oe.State == "cancelled" {
				return nil
			}
			return err
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("operation %s did not finish within %s (check it with `ca operation get %s`): %w",
				id, timeout, id, context.DeadlineExceeded)
		}
		return err
	}
	return a.printOperation(op)
}

const (
	pollStart = 2 * time.Second
	pollCap   = 10 * time.Second
)

// pollOperation reads the operation until it is complete (nil error), failed
// or cancelled (an *OperationError with the operation), or ctx ends.
func (a *App) pollOperation(ctx context.Context, s *session, id string) (*cloudafrica.Operation, error) {
	delay, ceiling := pollStart, pollCap
	if a.OperationPoll > 0 {
		delay, ceiling = a.OperationPoll, a.OperationPoll
	}
	last := ""
	for {
		op, err := getOperation(ctx, s, id)
		if err != nil {
			return nil, err
		}
		state := strings.ToLower(output.S(op.State))
		if a.TTY && !a.quiet {
			if p := progressLine(op); p != last {
				fmt.Fprintln(a.Stderr, p)
				last = p
			}
		}
		switch state {
		case "complete":
			return op, nil
		case "pending", "running":
		case "failed", "cancelled":
			oe := &cloudafrica.OperationError{ID: id, State: state}
			if op.Error != nil {
				oe.Kind = output.S(op.Error.Kind)
				oe.Message = output.S(op.Error.Message)
			}
			return op, oe
		default:
			return op, fmt.Errorf("operation %s has unexpected state %q", id, state)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
		delay = min(delay*2, ceiling)
	}
}

func progressLine(op *cloudafrica.Operation) string {
	line := output.S(op.Kind) + ": " + strings.ToLower(output.S(op.State))
	if op.ProgressPct != nil {
		line += fmt.Sprintf(" %d%%", *op.ProgressPct)
	}
	if m := output.S(op.ProgressMessage); m != "" {
		line += " " + m
	}
	return line
}

// accepted waits for a 202's operation, or prints it with --no-wait.
func (a *App) accepted(ctx context.Context, s *session, opID string, wf waitFlags) error {
	if opID == "" {
		return errors.New("the API accepted the request but returned no operation id")
	}
	if !wf.noWait {
		return a.waitAndPrint(ctx, s, opID, wf)
	}
	pr, err := a.printer()
	if err != nil {
		return err
	}
	if pr.Format == output.Table && !pr.Quiet {
		fmt.Fprintf(a.Stdout, "Accepted: operation %s\n", opID)
		return nil
	}
	return pr.Print(output.Result{Value: map[string]string{"operationId": opID}, IDs: []string{opID}})
}
