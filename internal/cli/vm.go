package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	cloudafrica "github.com/CloudAfrica/cloudafrica-go"

	"github.com/CloudAfrica/cloudafrica-cli/internal/output"
)

var vmHeaders = []string{"ID", "NAME", "STATUS", "VCPU", "RAM MIB", "DATACENTER", "PRIVATE IP", "PUBLIC IP"}

func vmRow(v cloudafrica.PublicVmDto) []string {
	cpu, mem := "", ""
	if v.Cpu != nil {
		cpu = output.N(v.Cpu.Cores)
	}
	if v.Memory != nil {
		mem = output.N(v.Memory.TotalMb)
	}
	return []string{output.S(v.Id), output.S(v.Name), output.S(v.Status), cpu, mem, output.S(v.Datacenter),
		output.S(v.PrivateIp), output.S(v.PublicIp)}
}

// accepted202 turns a generated 202 response into its operation id, or the
// API error. A body without the id falls back to the id in the Location
// header (/v1/operations/<id>, as a path or a URL).
func accepted202(h *http.Response, body []byte, op *cloudafrica.OperationAccepted) (string, error) {
	if err := cloudafrica.CheckResponse(h, body); err != nil {
		return "", err
	}
	if op != nil && op.OperationId != nil && *op.OperationId != "" {
		return *op.OperationId, nil
	}
	if id, ok := operationIDFromLocation(h.Header.Get("Location")); ok {
		return id, nil
	}
	return "", fmt.Errorf("HTTP %d with no operation id", h.StatusCode)
}

func operationIDFromLocation(loc string) (string, bool) {
	u, err := url.Parse(loc)
	if err != nil {
		return "", false
	}
	id, ok := strings.CutPrefix(u.Path, "/v1/operations/")
	if !ok || id == "" {
		return "", false
	}
	return id, true
}

func (a *App) vmCmd() *cobra.Command {
	cmd := group("vm", "Manage virtual machines")
	cmd.AddCommand(a.vmList(), a.vmGet(), a.vmCreate(), a.vmResize(), a.vmRename(), a.vmDelete())

	type action struct {
		use, short string
		call       func(ctx context.Context, c *cloudafrica.ClientWithResponses, id string, guestTimeout int32) (string, error)
		guest      bool
	}
	for _, act := range []action{
		{use: "start", short: "Start a VM", call: func(ctx context.Context, c *cloudafrica.ClientWithResponses, id string, _ int32) (string, error) {
			r, err := c.StartVmWithResponse(ctx, id)
			if err != nil {
				return "", err
			}
			return accepted202(r.HTTPResponse, r.Body, r.JSON202)
		}},
		{use: "shutdown", short: "Shut a VM down through its guest OS", guest: true, call: func(ctx context.Context, c *cloudafrica.ClientWithResponses, id string, gt int32) (string, error) {
			p := &cloudafrica.ShutdownVmParams{}
			if gt > 0 {
				p.Timeout = &gt
			}
			r, err := c.ShutdownVmWithResponse(ctx, id, p)
			if err != nil {
				return "", err
			}
			return accepted202(r.HTTPResponse, r.Body, r.JSON202)
		}},
		{use: "reboot", short: "Reboot a VM", call: func(ctx context.Context, c *cloudafrica.ClientWithResponses, id string, _ int32) (string, error) {
			r, err := c.RebootVmWithResponse(ctx, id)
			if err != nil {
				return "", err
			}
			return accepted202(r.HTTPResponse, r.Body, r.JSON202)
		}},
		{use: "force-stop", short: "Power a VM off without a guest shutdown", call: func(ctx context.Context, c *cloudafrica.ClientWithResponses, id string, _ int32) (string, error) {
			r, err := c.ForceStopVmWithResponse(ctx, id)
			if err != nil {
				return "", err
			}
			return accepted202(r.HTTPResponse, r.Body, r.JSON202)
		}},
	} {
		var wf waitFlags
		var guestTimeout int32
		c := &cobra.Command{
			Use:   act.use + " VM_ID",
			Short: act.short,
			Args:  exactArgs(1, "VM_ID"),
			RunE: func(cmd *cobra.Command, args []string) error {
				return a.withSession(cmd.Context(), func(s *session) error {
					opID, err := act.call(cmd.Context(), s.client.API, args[0], guestTimeout)
					if err != nil {
						return err
					}
					return a.accepted(cmd.Context(), s, opID, wf)
				})
			},
		}
		wf.bind(c, true)
		if act.guest {
			c.Flags().Int32Var(&guestTimeout, "guest-timeout", 0, "seconds the guest gets to shut down (server default 60)")
		}
		cmd.AddCommand(c)
	}
	return cmd
}

func (a *App) vmList() *cobra.Command {
	var slug string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List VMs",
		Args:  noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			pr, err := a.printer()
			if err != nil {
				return err
			}
			return a.withSession(cmd.Context(), func(s *session) error {
				vms := []cloudafrica.PublicVmDto{}
				var cursor *string
				limit := 200
				for {
					p := &cloudafrica.ListVmsParams{Cursor: cursor, Limit: &limit}
					if slug != "" {
						p.Slug = &slug
					}
					r, err := s.client.API.ListVmsWithResponse(cmd.Context(), p)
					if err != nil {
						return err
					}
					if err := cloudafrica.CheckResponse(r.HTTPResponse, r.Body); err != nil {
						return err
					}
					if r.JSON200 == nil {
						return errors.New("empty VM list response")
					}
					if r.JSON200.Data != nil {
						vms = append(vms, *r.JSON200.Data...)
					}
					if r.JSON200.NextCursor == nil || *r.JSON200.NextCursor == "" {
						break
					}
					cursor = r.JSON200.NextCursor
				}
				res := output.Result{Value: vms, Headers: vmHeaders}
				for _, v := range vms {
					res.Rows = append(res.Rows, vmRow(v))
					res.IDs = append(res.IDs, output.S(v.Id))
				}
				return pr.Print(res)
			})
		},
	}
	cmd.Flags().StringVar(&slug, "slug", "", "only the VM with this slug")
	return cmd
}

func (a *App) vmGet() *cobra.Command {
	return &cobra.Command{
		Use:   "get VM_ID",
		Short: "Show a VM",
		Args:  exactArgs(1, "VM_ID"),
		RunE: func(cmd *cobra.Command, args []string) error {
			pr, err := a.printer()
			if err != nil {
				return err
			}
			return a.withSession(cmd.Context(), func(s *session) error {
				r, err := s.client.API.GetVmWithResponse(cmd.Context(), args[0])
				if err != nil {
					return err
				}
				if err := cloudafrica.CheckResponse(r.HTTPResponse, r.Body); err != nil {
					return err
				}
				if r.JSON200 == nil {
					return errors.New("empty VM response")
				}
				return pr.Print(output.Result{Value: r.JSON200, Headers: vmHeaders,
					Rows: [][]string{vmRow(*r.JSON200)}, IDs: []string{output.S(r.JSON200.Id)}})
			})
		},
	}
}

func (a *App) vmCreate() *cobra.Command {
	var (
		req            cloudafrica.PublicCreateVmRequest
		ramGib         int64
		sshKeys        []string
		extraDisks     []int64
		placementGroup string
		reservedIP     string
		publicIP       bool
		wf             waitFlags
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a VM",
		Args:  noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := requireFlags(cmd, "name", "datacenter", "image", "vpc", "vcpu", "ram-gib", "boot-disk-gib"); err != nil {
				return err
			}
			if !req.TermsAccepted {
				return usageErr(errors.New("--accept-terms is required: creating a VM accepts the terms of use"))
			}
			req.RamMib = ramGib * 1024
			if len(sshKeys) > 0 {
				req.SshKeyIds = &sshKeys
			}
			if len(extraDisks) > 0 {
				req.ExtraDisksGib = &extraDisks
			}
			if placementGroup != "" {
				req.PlacementGroupId = &placementGroup
			}
			if reservedIP != "" {
				req.ReservedPublicIpId = &reservedIP
			}
			if cmd.Flags().Changed("public-ip") {
				req.PublicIp = &publicIP
			}
			return a.withSession(cmd.Context(), func(s *session) error {
				// The transport stamps a fresh Idempotency-Key on every write;
				// an empty param leaves that one in place.
				r, err := s.client.API.CreateVmWithResponse(cmd.Context(), &cloudafrica.CreateVmParams{}, req)
				if err != nil {
					return err
				}
				opID, err := accepted202(r.HTTPResponse, r.Body, r.JSON202)
				if err != nil {
					return err
				}
				return a.accepted(cmd.Context(), s, opID, wf)
			})
		},
	}
	f := cmd.Flags()
	f.StringVar(&req.Name, "name", "", "DNS-style name, unique in the account (required)")
	f.StringVar(&req.DatacenterSlug, "datacenter", "", "datacenter slug (required)")
	f.StringVar(&req.OsImageId, "image", "", "OS image id (required)")
	f.StringVar(&req.VpcResourceId, "vpc", "", "VPC id the VM joins (required)")
	f.Int32Var(&req.Vcpu, "vcpu", 0, "vCPU count, even (required)")
	f.Int64Var(&ramGib, "ram-gib", 0, "RAM in GiB, even (required)")
	f.Int64Var(&req.BootDiskGib, "boot-disk-gib", 0, "boot disk size in GiB (required)")
	f.Int64SliceVar(&extraDisks, "extra-disk-gib", nil, "extra data disk in GiB (repeatable, max 8)")
	f.StringSliceVar(&sshKeys, "ssh-key", nil, "SSH key id to install (repeatable)")
	f.BoolVar(&publicIP, "public-ip", false, "attach a public IP")
	f.StringVar(&reservedIP, "reserved-ip", "", "build onto this reserved public IP id")
	f.StringVar(&placementGroup, "placement-group", "", "placement group id")
	f.BoolVar(&req.TermsAccepted, "accept-terms", false, "accept the terms of use (required)")
	wf.bind(cmd, true)
	return cmd
}

func (a *App) vmResize() *cobra.Command {
	var (
		vcpu   int32
		ramGib int32
		wf     waitFlags
	)
	cmd := &cobra.Command{
		Use:   "resize VM_ID",
		Short: "Change a VM's vCPU count or memory",
		Args:  exactArgs(1, "VM_ID"),
		RunE: func(cmd *cobra.Command, args []string) error {
			var body cloudafrica.PublicResizeRequest
			if cmd.Flags().Changed("vcpu") {
				body.Cpus = &vcpu
			}
			if cmd.Flags().Changed("ram-gib") {
				mib := ramGib * 1024
				body.RamMib = &mib
			}
			if body.Cpus == nil && body.RamMib == nil {
				return usageErr(errors.New("give --vcpu, --ram-gib or both"))
			}
			return a.withSession(cmd.Context(), func(s *session) error {
				r, err := s.client.API.ResizeVmWithResponse(cmd.Context(), args[0], &cloudafrica.ResizeVmParams{}, body)
				if err != nil {
					return err
				}
				opID, err := accepted202(r.HTTPResponse, r.Body, r.JSON202)
				if err != nil {
					return err
				}
				return a.accepted(cmd.Context(), s, opID, wf)
			})
		},
	}
	cmd.Flags().Int32Var(&vcpu, "vcpu", 0, "new vCPU count, even")
	cmd.Flags().Int32Var(&ramGib, "ram-gib", 0, "new RAM in GiB, even")
	wf.bind(cmd, true)
	return cmd
}

func (a *App) vmRename() *cobra.Command {
	var wf waitFlags
	cmd := &cobra.Command{
		Use:   "rename VM_ID NEW_NAME",
		Short: "Rename a VM",
		Args:  exactArgs(2, "VM_ID NEW_NAME"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.withSession(cmd.Context(), func(s *session) error {
				r, err := s.client.API.RenameVmWithResponse(cmd.Context(), args[0], cloudafrica.PublicRenameRequest{Name: args[1]})
				if err != nil {
					return err
				}
				opID, err := accepted202(r.HTTPResponse, r.Body, r.JSON202)
				if err != nil {
					return err
				}
				return a.accepted(cmd.Context(), s, opID, wf)
			})
		},
	}
	wf.bind(cmd, true)
	return cmd
}

func (a *App) vmDelete() *cobra.Command {
	var (
		yes bool
		wf  waitFlags
	)
	cmd := &cobra.Command{
		Use:   "delete VM_ID",
		Short: "Delete a VM (a protected action: needs an elevated sign-in)",
		Args:  exactArgs(1, "VM_ID"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				if !a.TTY {
					return usageErr(errors.New("refusing to delete without --yes when not on a terminal"))
				}
				if !a.confirm(fmt.Sprintf("Delete VM %s? This cannot be undone.", args[0])) {
					return errors.New("cancelled")
				}
			}
			return a.withSession(cmd.Context(), func(s *session) error {
				r, err := s.client.API.DeleteVmWithResponse(cmd.Context(), args[0])
				if err != nil {
					return err
				}
				opID, err := accepted202(r.HTTPResponse, r.Body, r.JSON202)
				if err != nil {
					return err
				}
				return a.accepted(cmd.Context(), s, opID, wf)
			})
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "do not ask for confirmation")
	wf.bind(cmd, true)
	return cmd
}
