package diff

import (
	"fmt"
	"io"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
)

// Parse turns raw unified diff text into structured FileDiffs.
// Empty input returns an empty slice and no error.
func Parse(r io.Reader) ([]FileDiff, error) {
	files, _, err := gitdiff.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("invalid diff: %w", err)
	}

	out := make([]FileDiff, 0, len(files))
	for _, f := range files {
		if f.IsBinary {
			continue // nothing reviewable in a binary file
		}

		fd := FileDiff{
			OldPath:  f.OldName,
			NewPath:  f.NewName,
			IsNew:    f.IsNew,
			IsDelete: f.IsDelete,
			IsRename: f.IsRename,
		}
		// For deleted files NewName is empty; fall back so NewPath
		// always has the path to report against.
		if fd.NewPath == "" {
			fd.NewPath = f.OldName
		}

		for _, frag := range f.TextFragments {
			h := Hunk{
				OldStart: int(frag.OldPosition),
				OldCount: int(frag.OldLines),
				NewStart: int(frag.NewPosition),
				NewCount: int(frag.NewLines),
			}

			// This is the core of the step: walk the hunk keeping two
			// counters so every line knows its real line number.
			oldNo := int(frag.OldPosition)
			newNo := int(frag.NewPosition)

			for _, gl := range frag.Lines {
				content := strings.TrimSuffix(gl.Line, "\n")
				switch gl.Op {
				case gitdiff.OpContext:
					h.Lines = append(h.Lines, Line{
						Op: OpContext, Content: content,
						OldLine: oldNo, NewLine: newNo,
					})
					oldNo++
					newNo++
				case gitdiff.OpAdd:
					h.Lines = append(h.Lines, Line{
						Op: OpAdd, Content: content,
						NewLine: newNo,
					})
					newNo++
				case gitdiff.OpDelete:
					h.Lines = append(h.Lines, Line{
						Op: OpDelete, Content: content,
						OldLine: oldNo,
					})
					oldNo++
				}
			}

			fd.Hunks = append(fd.Hunks, h)
		}

		out = append(out, fd)
	}
	return out, nil
}
