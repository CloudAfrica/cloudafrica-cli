package cli

import (
	"strings"
	"testing"
)

const waits = "\n\nWaits for the operation to finish; --no-wait prints it and returns."

// The long help is the API description minus exactly the three API-only sentences, plus the wait line
// where the command starts an operation (CA-5660 §5).
func TestLongHelpKeepsWhatItShould(t *testing.T) {
	want := map[string]string{
		"deleteBucket":             "Irreversible: the objects go with the bucket, and the name is never reusable afterwards. Needs read:s3 as well as this route's own scope: the bucket is named by id, and the gateway resolves it through the account's gated bucket list, which is itself a read:s3 route. Requires protected:s3. The body must carry confirmBucketName, the bucket's name spelled exactly as the list returns it — the comparison is exact, matching the backend's own. Asynchronous: returns 202 with the operation to poll at Location, and 409 when a delete for this bucket is already in flight. 503 when the control plane is unavailable: the gateway renders it as its generic upstream-unavailable, an instruction to retry shortly." + waits,
		"addVmDisk":                "Billing-affecting: the new volume raises the VM's disk charge. Requires write:compute. Must carry an Idempotency-Key header: a retry under the same key replays the first call's stored response rather than starting a second, 409 while that first call is still in flight, and 422 if the key is reused for a different request. That matters most here: a blind retry without one adds a SECOND billable volume." + waits,
		"attachVmNetworkInterface": "Adds an interface in one of the account's VPCs; the platform picks its address. Requires protected:network. 202 does not mean the interface is up inside the guest: it is hot-plugged, and the guest's OS must configure it. Refuses with 404 for an unknown VM or VPC." + waits,
		"createVpcReservedIp":      "Requires write:network. Takes a free public address from the pool and holds it for the account until it is released; the address is not a parameter. Synchronous: returns 200 with the reservation. Refuses with 409 when no address is free, and with 422 limit-reached at the account's public-IP cap (the PUBLIC_IP key), which a reservation counts against whether or not it is attached. A retry with the same Idempotency-Key replays a 200 already returned; after a timeout or 5xx, or under a new key, it may reserve a second address, so list GET /v1/vpcs/{id}/reserved-ips first.",
		"listActivity":             "Two modes on one route. The account feed takes `types`, `cursor` and `limit` and returns the account's activity newest first; pass the previous response's nextCursor as `cursor` for the next page. The subject history takes `subjectType` and `subjectId` (and `limit`) and returns that one subject's newest activity, at most `limit` rows and never a nextCursor. Mixing the two is 400. `limit` is 1 to 50 and defaults to 50. Rows are limited to what the token's scopes could read directly, and rows show teammates' short display names, or \"Service token\" for an account service token. Readable by any token of the account; it needs no scope of its own.",
	}
	for _, op := range genOps {
		if w, ok := want[op.OperationID]; ok {
			if op.Long != w {
				t.Errorf("%s long help:\n got %q\nwant %q", op.OperationID, op.Long, w)
			}
			delete(want, op.OperationID)
		}
	}
	for id := range want {
		t.Errorf("%s is not a generated operation", id)
	}
}

// No long help keeps an API-only sentence, and every command that starts an operation says it waits.
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
		if got := strings.HasSuffix(op.Long, strings.TrimSpace(waits)); got != (op.Async != "") {
			t.Errorf("%s (async %q): ends with the wait line = %v", op.OperationID, op.Async, got)
		}
	}
}
