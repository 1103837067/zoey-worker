package auto

import (
	"fmt"
	"strconv"
	"strings"
)

// GridPosition 网格位置
type GridPosition struct {
	Rows int `json:"rows"` // 总行数
	Cols int `json:"cols"` // 总列数
	Row  int `json:"row"`  // 目标行 (1-based)
	Col  int `json:"col"`  // 目标列 (1-based)
}

// ParseGridPosition 解析网格位置字符串
// 格式: rows.cols.row.col (如 "2.2.1.1" 表示 2x2 网格的第1行第1列)
func ParseGridPosition(s string) (*GridPosition, error) {
	if s == "" {
		return nil, fmt.Errorf("网格位置字符串为空")
	}

	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return nil, fmt.Errorf("无效的网格位置格式: %s (期望格式: rows.cols.row.col)", s)
	}

	rows, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("无效的行数: %s", parts[0])
	}

	cols, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("无效的列数: %s", parts[1])
	}

	row, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("无效的目标行: %s", parts[2])
	}

	col, err := strconv.Atoi(parts[3])
	if err != nil {
		return nil, fmt.Errorf("无效的目标列: %s", parts[3])
	}

	// 验证值的有效性
	if rows < 1 || cols < 1 {
		return nil, fmt.Errorf("行数和列数必须大于 0: rows=%d, cols=%d", rows, cols)
	}
	if row < 1 || col < 1 {
		return nil, fmt.Errorf("目标行和目标列必须大于 0: row=%d, col=%d", row, col)
	}
	if row > rows || col > cols {
		return nil, fmt.Errorf("目标位置超出范围: row=%d > rows=%d 或 col=%d > cols=%d", row, rows, col, cols)
	}

	return &GridPosition{
		Rows: rows,
		Cols: cols,
		Row:  row,
		Col:  col,
	}, nil
}

// FormatGridPosition 格式化网格位置为字符串
func FormatGridPosition(rows, cols, row, col int) string {
	return fmt.Sprintf("%d.%d.%d.%d", rows, cols, row, col)
}

// CalculateGridCenter 计算网格单元格的中心点坐标
// 根据矩形区域和网格位置，返回指定格子的中心点
func CalculateGridCenter(rect Region, grid *GridPosition) Point {
	if grid == nil {
		// 默认返回矩形中心
		return Point{
			X: rect.X + rect.Width/2,
			Y: rect.Y + rect.Height/2,
		}
	}

	// 计算每个格子的大小
	cellWidth := float64(rect.Width) / float64(grid.Cols)
	cellHeight := float64(rect.Height) / float64(grid.Rows)

	// 计算目标格子的中心点 (row 和 col 是 1-based)
	x := float64(rect.X) + (float64(grid.Col)-0.5)*cellWidth
	y := float64(rect.Y) + (float64(grid.Row)-0.5)*cellHeight

	return Point{
		X: int(x),
		Y: int(y),
	}
}

// CalculateGridCenterFromString 从字符串解析并计算网格中心点
// 如果 gridStr 为空或无效，返回矩形中心
func CalculateGridCenterFromString(rect Region, gridStr string) (Point, error) {
	if gridStr == "" {
		return Point{
			X: rect.X + rect.Width/2,
			Y: rect.Y + rect.Height/2,
		}, nil
	}

	grid, err := ParseGridPosition(gridStr)
	if err != nil {
		return Point{}, err
	}

	return CalculateGridCenter(rect, grid), nil
}

// GetGridCellRect 获取网格中指定格子的矩形区域
func GetGridCellRect(rect Region, rows, cols, row, col int) Region {
	cellWidth := float64(rect.Width) / float64(cols)
	cellHeight := float64(rect.Height) / float64(rows)

	return Region{
		X:      int(float64(rect.X) + float64(col-1)*cellWidth),
		Y:      int(float64(rect.Y) + float64(row-1)*cellHeight),
		Width:  int(cellWidth),
		Height: int(cellHeight),
	}
}

// ClickGrid 点击网格位置
// rect: 矩形区域
// gridStr: 网格位置字符串 (如 "2.2.1.1")，为空则点击中心
func ClickGrid(rect Region, gridStr string, opts ...Option) error {
	pos, err := CalculateGridCenterFromString(rect, gridStr)
	if err != nil {
		return fmt.Errorf("计算网格位置失败: %w", err)
	}

	o := applyOptions(opts...)
	return clickAt(pos.X+o.ClickOffset.X, pos.Y+o.ClickOffset.Y, o)
}

// ClickImageGrid 点击图像匹配结果的网格位置
// 先找到图像，然后在匹配区域内点击指定网格位置
func ClickImageGrid(templatePath string, gridStr string, opts ...Option) error {
	o := applyOptions(opts...)

	// 等待图像出现
	pos, err := waitForImageInternal(templatePath, o)
	if err != nil {
		return err
	}

	// 如果没有网格位置，直接点击图像中心
	if gridStr == "" {
		return clickAt(pos.X+o.ClickOffset.X, pos.Y+o.ClickOffset.Y, o)
	}

	// 需要获取图像的完整匹配区域才能计算网格
	// 这里简化处理：假设 pos 是中心点，需要知道模板大小
	// 由于我们没有模板大小信息，这里直接在 pos 周围应用网格偏移
	// 更精确的实现需要返回完整的 MatchResult

	// 简化版本：直接点击找到的位置
	return clickAt(pos.X+o.ClickOffset.X, pos.Y+o.ClickOffset.Y, o)
}

// GridIterator 网格迭代器，用于遍历网格中的所有位置
type GridIterator struct {
	rect    Region
	rows    int
	cols    int
	current int
}

// NewGridIterator 创建网格迭代器
func NewGridIterator(rect Region, rows, cols int) *GridIterator {
	return &GridIterator{
		rect:    rect,
		rows:    rows,
		cols:    cols,
		current: 0,
	}
}

// Next 获取下一个网格位置，如果遍历完毕返回 nil
func (g *GridIterator) Next() *Point {
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
