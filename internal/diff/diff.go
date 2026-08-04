package diff

import "encoding/json"

// 0p represents what happened to a line: unchanged, added, or deleted
type Op int

const (
	OpContext Op = iota // unchanged, shown
	OpAdd               // added
	OpDelete            // deleted
)

// makes 0p show up as context/add/delete instead of 0/1/2
func (o Op) MarshalJSON() ([]byte, error) {
	switch o {
	case OpAdd:
		return json.Marshal("add")
	case OpDelete:
		return json.Marshal("delete")
	default:
		return json.Marshal("context")
	}
}

// one line of hunk w/ real line numbers
type Line struct {
	Op      Op
	Content string // without the leading +/-/space, without trailing \n
	OldLine int    // line num in old file; 0 when Op == OpAdd
	NewLine int    // line num in new file; 0 when Op == OpDelete
}

// one changed section of a file (one @@ ... @@ block)
type Hunk struct {
	OldStart, OldCount int
	NewStart, NewCount int
	Lines              []Line
}

// Only returns lines this hunk adds
func (h Hunk) AddedLines() []Line {
	var out []Line
	for _, l := range h.Lines {
		if l.Op == OpAdd {
			out = append(out, l)
		}
	}
	return out
}

// all changes to one file.
type FileDiff struct {
	OldPath  string
	NewPath  string
	IsNew    bool
	IsDelete bool
	IsRename bool
	Hunks    []Hunk
}
