// Package cmdspec is the shape of the generated command table: the generator
// (internal/gen) writes it and the executor (internal/cli) runs it.
package cmdspec

// Async is how a command's 202 names the operation to wait on.
type Async string

const (
	// NotAsync: the operation answers 200, 201 or 204.
	NotAsync Async = ""
	// Location: the 202 carries the operation id in its body or Location header.
	Location Async = "location"
	// Self: the 202 names nothing; wait on the operation the positional names.
	Self Async = "self"
)

// Op is one generated command.
type Op struct {
	// Path is the command path under `ca`, e.g. ["vm", "nic", "attach"].
	Path        []string
	OperationID string
	Method      string
	// URL is the path template, e.g. "/v1/vms/{id}/network-interfaces/{nicId}".
	URL     string
	Short   string
	Long    string
	Example string
	// Positional is the path parameter given as the argument; Name "" means none.
	Positional Param
	// Params are the other path parameters, each a required flag.
	Params []Param
	Query  []Field
	Body   []Field
	// BodyNested is true when the body has fields only --data can set.
	BodyNested bool
	// HasBody is true when the operation takes a JSON body.
	HasBody bool
	// List is true for a CursorPage list: it follows every page by default.
	List bool
	// Paged is true for a list that takes cursor and limit.
	Paged bool
	Async Async
	// Confirm is true for a delete or a protected action.
	Confirm bool
	// Columns names the column set for the 200/201 body ("" when none).
	Columns string
	// AtLeastOne lists flags of which at least one must be given.
	AtLeastOne []string
	Scopes     []string
}

// Param is a path parameter.
type Param struct {
	Name string
	Flag string
	// Resource names the Resource that resolves a name to an id ("" for ids only).
	Resource string
	Convert  string
	// Verbatim parameters (pattern admits "/") are joined unescaped.
	Verbatim bool
	Help     string
}

// Field is a query parameter or a top-level body field.
type Field struct {
	Name string
	Flag string
	// Type is string, integer, number or boolean.
	Type string
	// Format is the integer's width, int32 or int64 ("" for int64).
	Format   string
	Enum     []string
	Required bool
	// Scale multiplies an integer flag before it is sent (GiB → MiB is 1024).
	Scale int64
	Help  string
}

// Resource is how a name is resolved to an id.
type Resource struct {
	Noun string
	// List is the CursorPage list path.
	List string
	// Match is the field a name is compared with.
	Match string
	// ID is the field holding the id (default "id").
	ID string
	// IDPattern, when set, recognises an id without a lookup.
	IDPattern string
}

// ColumnSet is how a schema prints as a table.
type ColumnSet struct {
	// Rows is the JSON field holding a list's rows ("" means "data").
	Rows string
	// ID is the field -q prints.
	ID      string
	Columns []Column
}

// Column is one table column.
type Column struct {
	Header string
	// Path is a dotted JSON path, e.g. "cpu.cores".
	Path string
	// Format is "", bytes, time, upper, bool, list or count.
	Format string
}

// Group is a command group's help.
type Group struct {
	Path  []string
	Short string
}

// Covered lists an operation a hand-written command covers or an excluded one, with why.
type Covered struct {
	OperationID string
	Reason      string
}
