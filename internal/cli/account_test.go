package cli

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/CloudAfrica/cloudafrica-cli/internal/fakeapi"
)

func readURL(h *harness, want string) bool { return slices.Contains(h.fake.ReadURLs, want) }

// stamp is how a time column prints an RFC 3339 value in this environment's zone.
func stamp(rfc3339 string) string {
	t, err := time.Parse(time.RFC3339, rfc3339)
	if err != nil {
		panic(err)
	}
	return t.Local().Format("2006-01-02 15:04 MST")
}

// row is table()'s rendering of one row whose cells are given in order.
func row(cells ...string) string {
	return fmt.Sprint([][]string{strings.Fields(strings.Join(cells, " "))})
}

// Account reads print their own columns; a null limit is "unlimited" and a
// null usage stays blank.
func TestAccountReads(t *testing.T) {
	h := s3Harness(t)
	canned(h, map[string]fakeapi.Canned{
		"GET /v1/account": {Body: `{"accountId":"acc-1","accountName":"Globex","tokenName":"ci","expiresAt":"2026-12-01T08:00:00Z","scopes":["read:compute","write:compute"],
			"rateLimit":{"limit":600,"remaining":599,"resetAt":1}}`},
		"GET /v1/account/limits": {Body: `{"data":[
			{"key":"vm.count","scope":"ACCOUNT","limit":50,"used":7},
			{"key":"vm.disks","scope":"VM","limit":null,"used":null}]}`},
		"GET /v1/account/deleted-vm-backup-retention": {Body: `{"retentionDays":14}`},
	})
	head, rows := table(h, "account", "get")
	if head != "ACCOUNT ID ACCOUNT TOKEN EXPIRES SCOPES RATE LIMIT REMAINING" ||
		rows != row("acc-1", "Globex", "ci", stamp("2026-12-01T08:00:00Z"), "read:compute, write:compute", "600", "599") {
		t.Errorf("account get %q %s", head, rows)
	}
	head, rows = table(h, "account", "limits")
	if head != "KEY SCOPE LIMIT USED" || rows != "[[vm.count ACCOUNT 50 7] [vm.disks VM unlimited]]" {
		t.Errorf("account limits %q %s", head, rows)
	}
	head, rows = table(h, "account", "backup-retention")
	if head != "RETENTION DAYS" || rows != "[[14]]" {
		t.Errorf("backup-retention %q %s", head, rows)
	}
}

// --types repeats the query key once per value, whether given as commas or as
// a repeated flag, and an empty list or an empty value is refused before any
// request. The route caps a page at 50, so that is the page size asked for.
func TestActivityTypesRepeat(t *testing.T) {
	h := s3Harness(t)
	canned(h, map[string]fakeapi.Canned{"GET /v1/activity": {Body: `{"data":[
		{"id":"a-1","startedAt":"2026-11-22T08:00:00Z","verb":"vm.start","status":"failed","actorClass":"CUSTOMER","actorDisplayName":"Jo B",
		 "subjectType":"vm","subjectLabel":"web-01","failureReason":"quota"}]}`}})
	head, rows := table(h, "activity", "list", "--types", "vm,vpc", "--types", "ip")
	if head != "ID STARTED VERB STATUS ACTOR ACTOR NAME SUBJECT SUBJECT NAME REASON" ||
		rows != row("a-1", stamp("2026-11-22T08:00:00Z"), "vm.start", "failed", "CUSTOMER", "Jo B", "vm", "web-01", "quota") {
		t.Errorf("activity %q %s", head, rows)
	}
	if want := "GET /v1/activity?limit=50&types=vm&types=vpc&types=ip"; !readURL(h, want) {
		t.Errorf("reads %v, want %s", h.fake.ReadURLs, want)
	}
	h.mustRun("activity", "list", "--subject-type", "vm", "--subject-id", "vm-1", "-o", "json")
	if want := "GET /v1/activity?limit=50&subjectId=vm-1&subjectType=vm"; !readURL(h, want) {
		t.Errorf("reads %v, want %s", h.fake.ReadURLs, want)
	}
	n := len(h.fake.ReadURLs)
	for _, v := range []string{"", "a,,b", " "} {
		if code := h.run("activity", "list", "--types", v); code != ExitUsage || len(h.fake.ReadURLs) != n {
			t.Errorf("--types %q: exit %d, reads %d", v, code, len(h.fake.ReadURLs)-n)
		}
	}
}

// The two modes of /v1/activity do not mix: the API answers 400, so the CLI
// refuses before any request.
func TestActivityModesDoNotMix(t *testing.T) {
	h := s3Harness(t)
	for _, args := range [][]string{
		{"activity", "list", "--types", "vm", "--subject-type", "vm"},
		{"activity", "list", "--subject-id", "vm-1", "--types", "vm"},
		{"activity", "list", "--subject-type", "vm"},
		{"activity", "list", "--subject-id", "vm-1"},
		{"activity", "list", "--subject-type", "vm", "--subject-id", ""},
	} {
		if code := h.run(args...); code != ExitUsage || len(h.fake.ReadURLs) != 0 || h.stderr.Len() == 0 {
			t.Errorf("%v: exit %d, reads %d\n%s", args, code, len(h.fake.ReadURLs), h.stderr)
		}
	}
}

// --limit above the route's documented cap, and --cursor with the history flags,
// are refused before any request (the API answers 400 to both).
func TestActivityLimitAndCursorAreChecked(t *testing.T) {
	h := s3Harness(t)
	for _, c := range []struct {
		want string
		args []string
	}{
		{"at most 50", []string{"activity", "list", "--limit", "100"}},
		{"cannot be combined", []string{"activity", "list", "--limit", "5", "--cursor", "abc", "--subject-type", "vm", "--subject-id", "x"}},
	} {
		if code := h.run(c.args...); code != ExitUsage || len(h.fake.ReadURLs) != 0 || !strings.Contains(h.stderr.String(), c.want) {
			t.Errorf("%v: exit %d, reads %d\n%s", c.args, code, len(h.fake.ReadURLs), h.stderr)
		}
	}
}

// Every list walks at the page size its route documents: the generator reads
// "from 1 to N" off the limit parameter, so no list is asked for more than it takes.
func TestPageLimitComesFromTheSpec(t *testing.T) {
	var n int
	for _, op := range genOps {
		if !op.Paged {
			continue
		}
		n++
		if op.PageLimit < 1 || op.PageLimit > pageLimit {
			t.Errorf("%s: PageLimit %d, want 1..%d", strings.Join(op.Path, " "), op.PageLimit, pageLimit)
		}
		if op.OperationID == "listActivity" && op.PageLimit != 50 {
			t.Errorf("listActivity: PageLimit %d, want 50", op.PageLimit)
		}
	}
	if n == 0 {
		t.Fatal("no paged operation found")
	}
}

// A flag's help may not carry backticks: pflag would turn the first span into
// the value's name and print the rest literally.
func TestGeneratedHelpHasNoBackticks(t *testing.T) {
	h := s3Harness(t)
	for _, args := range [][]string{{"member", "create", "--help"}, {"member", "role", "set", "--help"}} {
		h.mustRun(args...)
		out := h.stdout.String()
		_, flags, _ := strings.Cut(out, "\nFlags:\n")
		if strings.Contains(flags, "`") {
			t.Errorf("%v flag help has a backtick:\n%s", args, flags)
		}
		h.stdout.Reset()
	}
}

// A group's one-line title is never an auto-derived "Manage <noun>s" that is wrong in English.
func TestGroupTitlesAreWritten(t *testing.T) {
	h := s3Harness(t)
	h.mustRun("billing", "--help")
	if out := h.stdout.String(); strings.Contains(out, "detailss") {
		t.Errorf("billing help:\n%s", out)
	}
}

// A member is named by email, the role is a positional-plus-flag write, and an
// unknown role is refused before any write.
func TestMemberCommands(t *testing.T) {
	h := s3Harness(t)
	canned(h, map[string]fakeapi.Canned{
		"GET /v1/members": {Body: `{"data":[
			{"userId":"u-1","email":"jo@example.com","firstName":"Jo","lastName":"B","role":"operator","pending":true},
			{"userId":"u-2","email":"sam@example.com","role":"owner","pending":false}]}`},
		"PUT /v1/members/u-1/role":                       {Status: 204},
		"DELETE /v1/members/u-1":                         {Status: 204},
		"POST /v1/members/u-1/actions/resend-invitation": {Status: 204},
		"POST /v1/members/u-1/actions/cancel-invitation": {Status: 204},
		"POST /v1/members":                               {Body: `{"userId":"u-3","alreadyRegistered":false}`},
	})
	head, rows := table(h, "member", "list")
	if head != "USER ID EMAIL FIRST NAME LAST NAME ROLE PENDING" ||
		rows != "[[u-1 jo@example.com Jo B operator yes] [u-2 sam@example.com owner no]]" {
		t.Errorf("member list %q %s", head, rows)
	}
	h.mustRun("member", "list", "-q")
	if got := strings.Fields(h.stdout.String()); !slices.Equal(got, []string{"u-1", "u-2"}) {
		t.Errorf("-q = %q", got)
	}
	h.mustRun("member", "role", "set", "jo@example.com", "--role", "admin", "--yes")
	h.mustRun("member", "resend-invitation", "jo@example.com")
	h.mustRun("member", "cancel-invitation", "u-1")
	h.mustRun("member", "delete", "jo@example.com", "--yes")
	h.mustRun("member", "create", "--email", "new@example.com", "--role", "viewer", "--first-name", "New", "--yes")
	h.stdout.Reset()
	h.mustRun("member", "create", "--email", "new@example.com", "--role", "viewer", "--yes", "-q")
	if got := strings.TrimSpace(h.stdout.String()); got != "u-3" {
		t.Errorf("member create -q = %q, want u-3", got)
	}
	want := []string{
		"PUT /v1/members/u-1/role", "POST /v1/members/u-1/actions/resend-invitation",
		"POST /v1/members/u-1/actions/cancel-invitation", "DELETE /v1/members/u-1", "POST /v1/members", "POST /v1/members",
	}
	if !slices.Equal(h.fake.WriteURLs, want) {
		t.Fatalf("writes %v", h.fake.WriteURLs)
	}
	if b := h.fake.Bodies[0]; b["role"] != "admin" || len(b) != 1 {
		t.Errorf("role body = %v", b)
	}
	if b := h.fake.Bodies[4]; b["email"] != "new@example.com" || b["role"] != "viewer" || b["firstName"] != "New" || len(b) != 3 {
		t.Errorf("create body = %v", b)
	}
	for _, args := range [][]string{
		{"member", "role", "set", "jo@example.com", "--role", "owner", "--yes"},
		{"member", "role", "set", "nobody@example.com", "--role", "admin", "--yes"},
		{"member", "create", "--email", "x@example.com", "--role", "owner", "--yes"},
	} {
		if code := h.run(args...); code != ExitUsage || len(h.fake.Writes) != len(want) {
			t.Errorf("%v: exit %d, writes %d", args, code, len(h.fake.Writes))
		}
	}
}

// A delete asks first: without --yes and without a terminal it refuses and sends nothing.
func TestAccountDeletesConfirm(t *testing.T) {
	h := s3Harness(t)
	canned(h, map[string]fakeapi.Canned{
		"GET /v1/members":         {Body: `{"data":[{"userId":"u-1","email":"jo@example.com"}]}`},
		"GET /v1/projects":        {Body: `{"data":[{"id":"p-1","name":"Prod"}]}`},
		"DELETE /v1/members/u-1":  {Status: 204},
		"DELETE /v1/projects/p-1": {Status: 204},
	})
	for _, args := range [][]string{{"member", "delete", "jo@example.com"}, {"project", "delete", "Prod"}} {
		if code := h.run(args...); code == 0 || len(h.fake.Writes) != 0 {
			t.Errorf("%v: exit %d, writes %d", args, code, len(h.fake.Writes))
		}
	}
}

// A project is named by name, create sends a key and the optional slug, a
// resource moves under the project's path, and a name two projects share is refused.
func TestProjectCommands(t *testing.T) {
	h := s3Harness(t)
	canned(h, map[string]fakeapi.Canned{
		"GET /v1/projects": {Body: `{"data":[
			{"id":"p-1","name":"Prod","slug":"PRD","defaultProject":false,"resourceCount":3,"createdAt":"2026-11-01T08:00:00Z"},
			{"id":"p-2","name":"Default","slug":"DEF","defaultProject":true,"resourceCount":0}]}`},
		"GET /v1/projects/p-1":                {Body: `{"id":"p-1","name":"Prod","slug":"PRD"}`},
		"GET /v1/projects/p-1/resources":      {Body: `{"data":[{"resourceId":"vm-1","type":"vm","name":"web-01","assignable":true},{"resourceId":"ip-1","type":"public_ip","name":"1.2.3.4","assignable":false}]}`},
		"POST /v1/projects":                   {Status: 201, Body: `{"id":"p-3","name":"Test","slug":"TST"}`},
		"PATCH /v1/projects/p-1":              {Body: `{"id":"p-1","name":"Production"}`},
		"PUT /v1/projects/p-1/resources/vm-1": {Body: `{"id":"p-1","name":"Prod"}`},
		"DELETE /v1/projects/p-1":             {Status: 204},
	})
	head, rows := table(h, "project", "list")
	if head != "ID NAME SLUG DEFAULT RESOURCES CREATED" || rows != fmt.Sprint([][]string{strings.Fields("p-1 Prod PRD no 3 " + stamp("2026-11-01T08:00:00Z")), {"p-2", "Default", "DEF", "yes", "0"}}) {
		t.Errorf("project list %q %s", head, rows)
	}
	head, rows = table(h, "project", "resource", "list", "--project", "Prod")
	if head != "ID TYPE NAME MOVABLE" || rows != "[[vm-1 vm web-01 yes] [ip-1 public_ip 1.2.3.4 no]]" {
		t.Errorf("resource list %q %s", head, rows)
	}
	h.mustRun("project", "get", "Prod", "-q")
	if got := strings.TrimSpace(h.stdout.String()); got != "p-1" {
		t.Errorf("get -q = %q", got)
	}
	h.mustRun("project", "create", "--name", "Test", "--slug", "TST")
	h.mustRun("project", "update", "Prod", "--name", "Production")
	h.mustRun("project", "resource", "move", "vm-1", "--project", "Prod")
	h.mustRun("project", "delete", "Prod", "--yes")
	want := []string{"POST /v1/projects", "PATCH /v1/projects/p-1", "PUT /v1/projects/p-1/resources/vm-1", "DELETE /v1/projects/p-1"}
	if !slices.Equal(h.fake.WriteURLs, want) {
		t.Fatalf("writes %v", h.fake.WriteURLs)
	}
	if b := h.fake.Bodies[0]; b["name"] != "Test" || b["slug"] != "TST" || len(b) != 2 {
		t.Errorf("create body = %v", b)
	}
	if h.fake.IdemKeys[0] == "" {
		t.Errorf("idempotency keys = %q", h.fake.IdemKeys)
	}
	if b := h.fake.Bodies[1]; b["name"] != "Production" || len(b) != 1 {
		t.Errorf("update body = %v", b)
	}
	canned(h, map[string]fakeapi.Canned{"GET /v1/projects": {Body: `{"data":[{"id":"p-1","name":"Prod"},{"id":"p-4","name":"Prod"}]}`}})
	if code := h.run("project", "get", "Prod"); code != ExitUsage || !strings.Contains(h.stderr.String(), "give the id instead") {
		t.Errorf("ambiguous name: exit %d\n%s", code, h.stderr)
	}
}

const ticketList = `{"data":[
	{"id":"t-1","reference":1234,"subject":"Backup failed","status":"open","raisedByName":"Jo B","updatedAt":"2026-11-20T09:30:00Z"},
	{"id":"t-2","reference":1235,"subject":"Invoice","status":"waiting-on-you","raisedByName":"Sam"}]}`

// A ticket is named by its reference number, the list filters by status as a
// repeated key, and create and reply send their bodies.
func TestSupportTicketCommands(t *testing.T) {
	h := s3Harness(t)
	canned(h, map[string]fakeapi.Canned{
		"GET /v1/support/tickets":                      {Body: ticketList},
		"GET /v1/support/tickets/t-1":                  {Body: `{"ticket":{"id":"t-1","reference":1234,"subject":"Backup failed","status":"open","raisedByName":"Jo B","updatedAt":"2026-11-20T09:30:00Z"},"descriptionHtml":"<p>Hi</p>","messages":[{"id":"m-1","body":"on it"},{"id":"m-2","body":"see you soon"}]}`},
		"POST /v1/support/tickets":                     {Status: 201, Body: `{"id":"t-3","reference":1236,"subject":"New","status":"open"}`},
		"POST /v1/support/tickets/t-1/replies":         {Body: `{"id":"t-1","reference":1234,"status":"open"}`},
		"POST /v1/support/tickets/t-1/actions/resolve": {Body: `{"id":"t-1","reference":1234,"status":"resolved"}`},
	})
	head, rows := table(h, "support", "ticket", "list", "--status", "open,waiting-on-you", "--search", "backup")
	if head != "REFERENCE ID SUBJECT STATUS RAISED BY UPDATED" ||
		rows != fmt.Sprint([][]string{strings.Fields("1234 t-1 Backup failed open Jo B " + stamp("2026-11-20T09:30:00Z")), {"1235", "t-2", "Invoice", "waiting-on-you", "Sam"}}) {
		t.Errorf("ticket list %q %s", head, rows)
	}
	if want := "GET /v1/support/tickets?limit=200&q=backup&status=open&status=waiting-on-you"; !readURL(h, want) {
		t.Errorf("reads %v, want %s", h.fake.ReadURLs, want)
	}
	h.mustRun("support", "ticket", "get", "1234", "-o", "table")
	if out := h.stdout.String(); !strings.Contains(out, "see you soon") || !strings.Contains(out, "Hi") {
		t.Errorf("ticket get hides the conversation:\n%s", out)
	}
	h.mustRun("support", "ticket", "get", "1234", "-q")
	if got := strings.TrimSpace(h.stdout.String()); got != "t-1" {
		t.Errorf("get -q = %q", got)
	}
	h.mustRun("support", "ticket", "create", "--subject", "New", "--description", "Help", "--requester-email", "jo@example.com")
	h.mustRun("support", "ticket", "reply", "1234", "--body", "Thanks")
	h.mustRun("support", "ticket", "resolve", "t-1")
	want := []string{"POST /v1/support/tickets", "POST /v1/support/tickets/t-1/replies", "POST /v1/support/tickets/t-1/actions/resolve"}
	if !slices.Equal(h.fake.WriteURLs, want) {
		t.Fatalf("writes %v", h.fake.WriteURLs)
	}
	if b := h.fake.Bodies[0]; b["subject"] != "New" || b["description"] != "Help" || b["requesterEmail"] != "jo@example.com" || len(b) != 3 {
		t.Errorf("create body = %v", b)
	}
	if b := h.fake.Bodies[1]; b["body"] != "Thanks" || len(b) != 1 {
		t.Errorf("reply body = %v", b)
	}
	if h.fake.IdemKeys[0] == "" {
		t.Errorf("idempotency keys = %q", h.fake.IdemKeys)
	}
	if code := h.run("support", "ticket", "get", "9999"); code != ExitUsage || len(h.fake.Writes) != len(want) {
		t.Errorf("unknown reference: exit %d", code)
	}
}

// Billing reads print their columns, details set sends only what was given
// and refuses to send nothing.
func TestBillingCommands(t *testing.T) {
	h := s3Harness(t)
	canned(h, map[string]fakeapi.Canned{
		"GET /v1/billing/balance": {Body: `{"current":"100.00","overdue":"0.00","total":"100.00","currency":"ZAR","computedAt":"2026-11-22T06:00:00Z"}`},
		"GET /v1/billing/accrual": {Body: `{"asOf":"2026-11-22T07:00:00Z","periodStart":"2026-11-01T00:00:00Z","periodEnd":"2026-12-01T00:00:00Z","total":"12.50","currency":"ZAR","billableHours":10.5,"unpriceableHours":0}`},
		"GET /v1/billing/details": {Body: `{"legalName":"Globex","billingEmail":"a@example.com","vatNumber":"4","companyNumber":"2020/1","postalAddress1":"1 Main","postalAddress2":"Floor 2","postalAddress3":"Sandton","postalAddress4":"Gauteng","postalAddress5":"2196","countryCode":"ZA","editable":true}`},
		"PUT /v1/billing/details": {Status: 204},
	})
	head, rows := table(h, "billing", "balance")
	if head != "CURRENT OVERDUE TOTAL CURRENCY COMPUTED" || rows != row("100.00", "0.00", "100.00", "ZAR", stamp("2026-11-22T06:00:00Z")) {
		t.Errorf("balance %q %s", head, rows)
	}
	head, rows = table(h, "billing", "accrual", "--as-of", "2026-11-01T00:00:00Z")
	if head != "AS OF PERIOD START PERIOD END TOTAL CURRENCY BILLABLE H UNPRICED H" || rows != row(stamp("2026-11-22T07:00:00Z"), stamp("2026-11-01T00:00:00Z"), stamp("2026-12-01T00:00:00Z"), "12.50", "ZAR", "10.5", "0") {
		t.Errorf("accrual %q %s", head, rows)
	}
	if want := "GET /v1/billing/accrual?asOf=2026-11-01T00%3A00%3A00Z"; !readURL(h, want) {
		t.Errorf("reads %v, want %s", h.fake.ReadURLs, want)
	}
	head, rows = table(h, "billing", "details", "get")
	if head != "LEGAL NAME EMAIL VAT COMPANY NO ADDRESS 1 ADDRESS 2 ADDRESS 3 ADDRESS 4 ADDRESS 5 COUNTRY EDITABLE" ||
		rows != row("Globex", "a@example.com", "4", "2020/1", "1 Main", "Floor 2", "Sandton", "Gauteng", "2196", "ZA", "yes") {
		t.Errorf("details %q %s", head, rows)
	}
	if code := h.run("billing", "details", "set", "--yes"); code != ExitUsage || len(h.fake.Writes) != 0 {
		t.Errorf("nothing to set: exit %d, writes %d", code, len(h.fake.Writes))
	}
	h.mustRun("billing", "details", "set", "--billing-email", "b@example.com", "--yes")
	if b := h.fake.Bodies[0]; b["billingEmail"] != "b@example.com" || len(b) != 1 {
		t.Errorf("body = %v", b)
	}
}

// Transactions, backup policies, backup windows and the feasibility check.
func TestAccountingAndPlacementReads(t *testing.T) {
	h := s3Harness(t)
	canned(h, map[string]fakeapi.Canned{
		"GET /v1/transactions": {Body: `{"data":[
			{"kind":"INVOICE","number":"INV-1","date":"2026-10-31","amount":"115.00","currency":"ZAR","status":"paid"},
			{"kind":"CREDIT_NOTE","number":"CN-1","date":"2026-11-02","amount":"-10.00","currency":"ZAR","status":"ISSUED","relatedNumber":"INV-1"}]}`},
		"GET /v1/backup-policies":       {Body: `{"data":[{"vmId":"vm-1","policy":{"planDisplayName":"Daily","cadenceKind":"weekly","cadenceDay":3,"windowPreference":"overnight","paused":false,"lastFiredSlot":"2026-11-21T22:30:00Z"}}]}`},
		"GET /v1/backup-windows":        {Body: `{"data":[{"preference":"overnight","startMinuteOfDayUtc":1350,"lengthMinutes":240}]}`},
		"GET /v1/placement/feasibility": {Body: `{"datacenterSlug":"jnb","requested":2,"fits":1,"available":true,"reasons":["ram"]}`},
	})
	head, rows := table(h, "transaction", "list")
	if head != "NUMBER KIND DATE AMOUNT CURRENCY STATUS RELATED" ||
		rows != "[[INV-1 INVOICE 2026-10-31 115.00 ZAR paid] [CN-1 CREDIT_NOTE 2026-11-02 -10.00 ZAR ISSUED INV-1]]" {
		t.Errorf("transactions %q %s", head, rows)
	}
	head, rows = table(h, "backup-policy", "list")
	if head != "VM PLAN CADENCE DAY WINDOW PAUSED LAST RUN" || rows != row("vm-1", "Daily", "weekly", "3", "overnight", "no", stamp("2026-11-21T22:30:00Z")) {
		t.Errorf("policies %q %s", head, rows)
	}
	head, rows = table(h, "backup-window", "list")
	if head != "WINDOW STARTS UTC LENGTH MIN" || rows != "[[overnight 22:30 240]]" {
		t.Errorf("windows %q %s", head, rows)
	}
	head, rows = table(h, "datacenter", "feasibility", "--datacenter", "jnb", "--vcpu", "4", "--ram-gib", "8", "--boot-disk-gib", "40", "--count", "2")
	if head != "DATACENTER AVAILABLE FITS REASONS" || rows != "[[jnb yes 1 ram]]" {
		t.Errorf("feasibility %q %s", head, rows)
	}
	if want := "GET /v1/placement/feasibility?bootDiskGib=40&count=2&datacenterSlug=jnb&ramMib=8192&vcpu=4"; !readURL(h, want) {
		t.Errorf("reads %v, want %s", h.fake.ReadURLs, want)
	}
	if code := h.run("datacenter", "feasibility", "--datacenter", "jnb", "--vcpu", "4", "--boot-disk-gib", "40"); code != ExitUsage {
		t.Errorf("without --ram-gib: exit %d", code)
	}
}
