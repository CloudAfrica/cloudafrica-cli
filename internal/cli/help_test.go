package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/CloudAfrica/cloudafrica-cli/internal/cmdspec"
)

const waits = "\n\n" + cmdspec.WaitLine

// The long help is the overlay's long: where commands.yaml sets one, else the API description minus the
// API-only sentences, plus the wait line where the command starts an operation (CA-5660 §5).
func TestLongHelpKeepsWhatItShould(t *testing.T) {
	want := map[string]string{
		"deleteBucket":             "Deletes the bucket and every object in it. It cannot be undone, and the name can never be used again. --confirm-bucket-name must spell the bucket's name exactly as `ca buckets list` shows it. A second delete while one is running is refused. Needs read:s3 as well, because ca looks the bucket up in your bucket list. Needs protected:s3." + waits,
		"addVmDisk":                "Attaches a new data disk, which adds to the VM's disk charge. If ca fails without an answer, run `ca vms disks list` before trying again: a second run adds a second billable disk. Needs write:compute." + waits,
		"attachVmNetworkInterface": "Adds a network interface in one of the account's VPCs; the platform picks its address. The interface is hot-plugged, so the guest's OS must still configure it. Needs protected:network." + waits,
		"createVpcReservedIp":      "Reserves a free public address from the pool for the account until you release it. It counts against the account's public-IP limit whether or not it is attached. It is refused when no address is free. If ca fails without an answer, run `ca vpcs reserved-ips list` before trying again: a second run can reserve a second address. Needs write:network.",
		"listActivity":             "Lists the account's activity, newest first. --types narrows it to kinds of resource. --subject-type with --subject-id shows one resource's newest activity instead, at most 50 rows and in one page. The two do not mix. You see the rows your scopes could read directly; teammates show by display name, and an account service token as \"Service token\". Needs no scope of its own.",
	}
	for _, op := range genOps {
		if w, ok := want[op.OperationID]; ok {
			if got := longHelp(op); got != w {
				t.Errorf("%s long help:\n got %q\nwant %q", op.OperationID, got, w)
			}
			delete(want, op.OperationID)
		}
	}
	for id := range want {
		t.Errorf("%s is not a generated operation", id)
	}
}

// No long help keeps an API-only sentence.
func TestLongHelpDropsTheAPIOnlySentences(t *testing.T) {
	for _, op := range genOps {
		for _, s := range []string{
			"Cursor-paginated: follow nextCursor via `?cursor=`.",
			"Must carry an Idempotency-Key header.",
			"Asynchronous: returns 202 with the operation to poll at Location.",
		} {
			if strings.Contains(op.Long, s) {
				t.Errorf("%s long help still says %q", op.OperationID, s)
			}
		}
	}
}

// Every command that waits, hand-written ones included, says so: its long help ends with the wait line
// exactly when it has --no-wait.
func TestEveryWaitingCommandSaysItWaits(t *testing.T) {
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		waits := c.LocalNonPersistentFlags().Lookup("no-wait") != nil
		if got := strings.HasSuffix(c.Long, cmdspec.WaitLine); got != waits {
			t.Errorf("%s (--no-wait %v): long help ends with the wait line = %v", c.CommandPath(), waits, got)
		}
		for _, s := range c.Commands() {
			walk(s)
		}
	}
	walk(treeRoot())
}

// Every protected: scope an operation needs is named in its help, so a user can tell which pairing to ask for.
func TestLongHelpNamesEveryProtectedScope(t *testing.T) {
	checked := 0
	for _, op := range genOps {
		for _, s := range op.Scopes {
			if !strings.HasPrefix(s, "protected:") {
				continue
			}
			checked++
			if !strings.Contains(longHelp(op), s) {
				t.Errorf("%s needs %s but its help does not say so", op.OperationID, s)
			}
		}
	}
	if checked < 30 {
		t.Fatalf("checked %d protected scopes: the walk is wrong", checked)
	}
}
