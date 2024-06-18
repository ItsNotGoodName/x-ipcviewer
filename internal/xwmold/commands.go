package xwmold

import (
	"math"
)

type LayoutGrid struct {
	width      uint16
	height     uint16
	paneWidth  uint16
	paneHeight uint16
	columns    int
	rows       int
}

func NewLayoutGrid(width, height uint16, count int) LayoutGrid {
	columns, rows := 0, 0
	for columns*rows < count {
		columns++
		if columns*rows >= count {
			break
		}
		rows++
	}
	paneWidth := uint16(float32(width) * (1.0 / float32(columns)))
	paneHeight := uint16(float32(height) * (1.0 / float32(rows)))
	return LayoutGrid{
		width:      width,
		height:     height,
		paneWidth:  paneWidth,
		paneHeight: paneHeight,
		columns:    columns,
		rows:       rows,
	}
}

func (l LayoutGrid) Pane(index int) (x int16, y int16, w uint16, h uint16) {
	row, col := math.Floor(float64(index/l.rows)), index%l.columns

	x = int16(l.paneWidth * uint16(col))
	y = int16(l.paneHeight * uint16(row))
	w = l.paneWidth
	h = l.paneHeight
	return
}