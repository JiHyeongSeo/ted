// Package conflict provides utilities for detecting and resolving git merge conflicts.
package conflict

import "strings"

// Block represents a single git conflict block in a file.
type Block struct {
	OursStart int // line index of "<<<<<<< HEAD"
	Sep       int // line index of "======="
	TheirsEnd int // line index of ">>>>>>> branch"
}

// Parse finds all conflict blocks in the given lines.
func Parse(lines []string) []Block {
	var blocks []Block
	i := 0
	for i < len(lines) {
		if strings.HasPrefix(lines[i], "<<<<<<<") {
			start := i
			sep := -1
			end := -1
			for j := i + 1; j < len(lines); j++ {
				if strings.HasPrefix(lines[j], "=======") && sep == -1 {
					sep = j
				} else if strings.HasPrefix(lines[j], ">>>>>>>") {
					end = j
					break
				}
			}
			if sep >= 0 && end > sep {
				blocks = append(blocks, Block{OursStart: start, Sep: sep, TheirsEnd: end})
				i = end + 1
				continue
			}
		}
		i++
	}
	return blocks
}

// HasConflicts reports whether the given lines contain any conflict markers.
func HasConflicts(lines []string) bool {
	for _, l := range lines {
		if strings.HasPrefix(l, "<<<<<<<") {
			return true
		}
	}
	return false
}

// Resolve replaces the conflict block in lines with the chosen side.
// choice: "ours", "theirs", "both"
// Returns new lines with the conflict markers removed.
func Resolve(lines []string, block Block, choice string) []string {
	var kept []string
	switch choice {
	case "ours":
		kept = lines[block.OursStart+1 : block.Sep]
	case "theirs":
		kept = lines[block.Sep+1 : block.TheirsEnd]
	case "both":
		kept = append(append([]string{}, lines[block.OursStart+1:block.Sep]...), lines[block.Sep+1:block.TheirsEnd]...)
	default:
		return lines
	}

	result := make([]string, 0, len(lines)-(block.TheirsEnd-block.OursStart+1)+len(kept))
	result = append(result, lines[:block.OursStart]...)
	result = append(result, kept...)
	result = append(result, lines[block.TheirsEnd+1:]...)
	return result
}

// BlockAt returns the conflict block that contains lineIdx, or nil if none.
func BlockAt(blocks []Block, lineIdx int) *Block {
	for i := range blocks {
		b := &blocks[i]
		if lineIdx >= b.OursStart && lineIdx <= b.TheirsEnd {
			return b
		}
	}
	return nil
}
