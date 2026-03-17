package git

// GraphCell represents a cell type in the graph visualization.
type GraphCell int

const (
	CellEmpty       GraphCell = iota // blank
	CellCommit                       // ● commit node
	CellPipe                         // │ vertical continuation
	CellMergeRight                   // ┘ merge/converge from right to left
	CellBranchRight                  // ┐ branch from left to right
	CellHorizontal                   // ─ horizontal connector
)

// GraphRow represents one row of the commit graph.
type GraphRow struct {
	Commit *Commit
	Cells  []GraphCell // one per column
	Colors []int       // color index per column
	Column int         // which column this commit is in
}

// BranchColorCount is the number of distinct branch colors.
const BranchColorCount = 8

// LayoutGraph assigns columns and connection cells to each commit.
//
// Algorithm overview (newest-first commit list):
//  - activeLanes[j] = hash of the commit we expect to see next in lane j
//  - When processing commit C:
//    1. Find ALL lanes whose expected hash == C.Hash  → matchingLanes
//    2. Primary column = matchingLanes[0] (leftmost), or a new lane if none found
//    3. For each extra matching lane: draw a right-to-left convergence (┘) at that lane and clear it
//       This fixes "ghost lanes" where multiple lanes pointed to the same parent commit
//    4. Set primary lane to C's first parent; add new lanes for merge parents
func LayoutGraph(commits []Commit) []GraphRow {
	if len(commits) == 0 {
		return nil
	}

	var activeLanes []string
	var laneColors []int
	colorCounter := 0

	rows := make([]GraphRow, 0, len(commits))

	for i := range commits {
		c := &commits[i]

		// --- Step 1: find ALL lanes matching this commit's hash ---
		var matchingLanes []int
		for j, h := range activeLanes {
			if h == c.Hash {
				matchingLanes = append(matchingLanes, j)
			}
		}

		// --- Step 2: determine (or create) primary column ---
		var col int
		if len(matchingLanes) == 0 {
			// Branch tip not yet seen: assign a new or recycled lane
			col = findEmptyLane(activeLanes)
			if col == -1 {
				col = len(activeLanes)
				activeLanes = append(activeLanes, c.Hash)
				laneColors = append(laneColors, colorCounter%BranchColorCount)
				colorCounter++
			} else {
				activeLanes[col] = c.Hash
				laneColors[col] = colorCounter % BranchColorCount
				colorCounter++
			}
		} else {
			col = matchingLanes[0] // leftmost is primary
		}

		// --- Step 3: build cell array ---
		numCols := len(activeLanes)
		cells := make([]GraphCell, numCols)
		colors := make([]int, numCols)
		copy(colors, laneColors)

		// All active lanes get a pipe; commit column gets the node
		for j := range activeLanes {
			if activeLanes[j] != "" && j != col {
				cells[j] = CellPipe
			}
		}
		cells[col] = CellCommit

		// --- Step 4: draw convergence for extra matching lanes ---
		// When multiple lanes were all waiting for this commit, they converge here.
		// Each extra lane (always to the right of `col` since matchingLanes is sorted) gets
		// a ┘ indicator and any lanes between get ─ (overwriting the pipe if needed).
		for mi := 1; mi < len(matchingLanes); mi++ {
			extraLane := matchingLanes[mi]
			// extraLane > col always (matchingLanes is in ascending order)
			for k := col + 1; k < extraLane; k++ {
				cells[k] = CellHorizontal
			}
			cells[extraLane] = CellMergeRight // ┘
			activeLanes[extraLane] = ""
		}

		// --- Step 5: advance primary lane to next expected commit ---
		if len(c.Parents) == 0 {
			activeLanes[col] = ""
		} else {
			activeLanes[col] = c.Parents[0]

			// Merge parents (parents[1+]): open new lanes or draw merge connectors
			for p := 1; p < len(c.Parents); p++ {
				parentHash := c.Parents[p]
				parentLane := -1
				for j, h := range activeLanes {
					if h == parentHash {
						parentLane = j
						break
					}
				}
				if parentLane == -1 {
					// New lane for this merge parent
					parentLane = findEmptyLane(activeLanes)
					if parentLane == -1 {
						parentLane = len(activeLanes)
						activeLanes = append(activeLanes, parentHash)
						laneColors = append(laneColors, colorCounter%BranchColorCount)
						colorCounter++
					} else {
						activeLanes[parentLane] = parentHash
						laneColors[parentLane] = colorCounter % BranchColorCount
						colorCounter++
					}

					for len(cells) < len(activeLanes) {
						cells = append(cells, CellEmpty)
						colors = append(colors, 0)
					}
					copy(colors, laneColors)

					if parentLane > col {
						for k := col + 1; k < parentLane; k++ {
							if cells[k] == CellEmpty {
								cells[k] = CellHorizontal
							}
						}
						cells[parentLane] = CellBranchRight
					}
				} else {
					if parentLane > col {
						for k := col + 1; k < parentLane; k++ {
							if cells[k] == CellEmpty {
								cells[k] = CellHorizontal
							}
						}
						if cells[parentLane] == CellPipe {
							cells[parentLane] = CellMergeRight
						}
					}
				}
			}
		}

		// Trim trailing empty lanes
		for len(activeLanes) > 0 && activeLanes[len(activeLanes)-1] == "" {
			activeLanes = activeLanes[:len(activeLanes)-1]
			laneColors = laneColors[:len(laneColors)-1]
		}

		rows = append(rows, GraphRow{
			Commit: c,
			Cells:  cells,
			Colors: colors,
			Column: col,
		})
	}

	return rows
}

func findEmptyLane(lanes []string) int {
	for i, h := range lanes {
		if h == "" {
			return i
		}
	}
	return -1
}
