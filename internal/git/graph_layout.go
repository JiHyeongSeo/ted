package git

// GraphCell represents a cell type in the graph visualization.
type GraphCell int

const (
	CellEmpty       GraphCell = iota // blank
	CellCommit                       // ● commit node
	CellPipe                         // │ vertical continuation
	CellMergeRight                   // ┘ merge from right to left
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

		col := -1
		for j, h := range activeLanes {
			if h == c.Hash {
				col = j
				break
			}
		}

		if col == -1 {
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
		}

		numCols := len(activeLanes)
		cells := make([]GraphCell, numCols)
		colors := make([]int, numCols)
		copy(colors, laneColors)

		for j := range activeLanes {
			if activeLanes[j] != "" && j != col {
				cells[j] = CellPipe
			}
		}
		cells[col] = CellCommit

		if len(c.Parents) == 0 {
			activeLanes[col] = ""
		} else {
			activeLanes[col] = c.Parents[0]

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
