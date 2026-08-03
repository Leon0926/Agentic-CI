package diff

type Op int

const (
	OpContext Op = iota
	OpAdd
	OpDelete
)

type Line struct {
	Op      Op
	Content string // without the leading +/-/space, without trailing \n
	OldLine int    // 0 when Op == OpAdd
	NewLine int    // 0 when Op == OpDelete
}

type Hunk struct {
	OldStart, OldCount int
	NewStart, NewCount int
	Lines              []Line
}

type FileDiff struct {
	OldPath  string
	NewPath  string
	IsNew    bool
	IsDelete bool
	IsRename bool
	Hunks    []Hunk
}

func (h Hunk) Addedlines() []Line
