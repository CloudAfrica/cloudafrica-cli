package cli

import (
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	cloudafrica "github.com/CloudAfrica/cloudafrica-go"

	"github.com/CloudAfrica/cloudafrica-cli/internal/cmdspec"
)

// vmCmd is the hand-written part of `ca vms`; the generated commands join it.
func (a *App) vmCmd() *cobra.Command {
	cmd := group("vms", "Manage virtual machines")
	cmd.AddCommand(a.vmCreate())
	return cmd
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
		Long:  "Creates a VM from flags; it takes no --data.\n\n" + cmdspec.WaitLine,
		Args:  noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := requireFlags(cmd, "name", "datacenter", "image", "vpc", "vcpu", "ram-gib", "boot-disk-gib"); err != nil {
				return err
			}
			if !req.TermsAccepted {
				return usageErr(errors.New("--accept-terms is required: creating a VM accepts the terms of use"))
			}
			if ramGib <= 0 || ramGib > math.MaxInt32/1024 {
				return usageErr(fmt.Errorf("--ram-gib %d is out of range", ramGib))
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
			// Names resolve in a reads-only session, so a step-up retry of the
			// create does not look them up again.
			err := a.withSession(cmd.Context(), func(s *session) error {
				var err error
				if req.VpcResourceId, _, err = a.resolveName(cmd.Context(), s, "vpc", req.VpcResourceId); err != nil {
					return err
				}
				for i := range sshKeys {
					if sshKeys[i], _, err = a.resolveName(cmd.Context(), s, "ssh-key", sshKeys[i]); err != nil {
						return err
					}
				}
				if placementGroup != "" {
					if placementGroup, _, err = a.resolveName(cmd.Context(), s, "placement-group", placementGroup); err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				return err
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
	f.StringVar(&req.DatacenterSlug, "datacenter", "", "datacenter slug, from `ca datacenters list` (required)")
	f.StringVar(&req.OsImageId, "image", "", "OS image id (required)")
	f.StringVar(&req.VpcResourceId, "vpc", "", "VPC id or name the VM joins, from `ca vpcs list` (required)")
	f.Int32Var(&req.Vcpu, "vcpu", 0, "vCPU count, even (required)")
	f.Int64Var(&ramGib, "ram-gib", 0, "RAM in GiB, even (required)")
	f.Int64Var(&req.BootDiskGib, "boot-disk-gib", 0, "boot disk size in GiB (required)")
	f.Int64SliceVar(&extraDisks, "extra-disk-gib", nil, "extra data disk in GiB (repeatable, max 8)")
	f.StringSliceVar(&sshKeys, "ssh-key", nil, "SSH key id or name to install, from `ca ssh-keys list` (repeatable)")
	f.BoolVar(&publicIP, "public-ip", false, "attach a public IP")
	f.StringVar(&reservedIP, "reserved-ip", "", "build onto this reserved public IP id")
	f.StringVar(&placementGroup, "placement-group", "", "placement group id or name")
	f.BoolVar(&req.TermsAccepted, "accept-terms", false, "accept the terms of use (required)")
	wf.bind(cmd, true)
	return cmd
}
