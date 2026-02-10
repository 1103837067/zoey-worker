package grid

import "github.com/zoeyai/zoeyworker/pkg/auto"

// GridIterator 网格迭代器，用于遍历网格中的所有位置
type GridIterator struct {
	rect    auto.Region
	rows    int
	cols    int
	current int
}

// NewGridIterator 创建网格迭代器
func NewGridIterator(rect auto.Region, rows, cols int) *GridIterator {
	return &GridIterator{
		rect:    rect,
		rows:    rows,
		cols:    cols,
		current: 0,
	}
}

// Next 获取下一个网格位置，如果遍历完毕返回 nil
func (g *GridIterator) Next() *auto.Point {
	if g.current >= g.rows*g.cols {
		return nil
	}

	row := g.current/g.cols + 1
	col := g.current%g.cols + 1
	g.current++

	grid := &GridPosition{
		Rows: g.rows,
		Cols: g.cols,
		Row:  row,
		Col:  col,
	}

	pos := CalculateGridCenter(g.rect, grid)
	return &pos
}

// Reset 重置迭代器
func (g *GridIterator) Reset() {
	g.current = 0
}

// Count 返回总格子数
func (g *GridIterator) Count() int {
	return g.rows * g.cols
}
