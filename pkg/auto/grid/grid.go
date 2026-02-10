// Package grid 提供网格计算和网格点击功能
package grid

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zoeyai/zoeyworker/pkg/auto"
	"github.com/zoeyai/zoeyworker/pkg/auto/input"
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
func CalculateGridCenter(rect auto.Region, grid *GridPosition) auto.Point {
	if grid == nil {
		return auto.Point{
			X: rect.X + rect.Width/2,
			Y: rect.Y + rect.Height/2,
		}
	}

	cellWidth := float64(rect.Width) / float64(grid.Cols)
	cellHeight := float64(rect.Height) / float64(grid.Rows)

	x := float64(rect.X) + (float64(grid.Col)-0.5)*cellWidth
	y := float64(rect.Y) + (float64(grid.Row)-0.5)*cellHeight

	return auto.Point{
		X: int(x),
		Y: int(y),
	}
}

// CalculateGridCenterFromString 从字符串解析并计算网格中心点
func CalculateGridCenterFromString(rect auto.Region, gridStr string) (auto.Point, error) {
	if gridStr == "" {
		return auto.Point{
			X: rect.X + rect.Width/2,
			Y: rect.Y + rect.Height/2,
		}, nil
	}

	grid, err := ParseGridPosition(gridStr)
	if err != nil {
		return auto.Point{}, err
	}

	return CalculateGridCenter(rect, grid), nil
}

// GetGridCellRect 获取网格中指定格子的矩形区域
func GetGridCellRect(rect auto.Region, rows, cols, row, col int) auto.Region {
	cellWidth := float64(rect.Width) / float64(cols)
	cellHeight := float64(rect.Height) / float64(rows)

	return auto.Region{
		X:      int(float64(rect.X) + float64(col-1)*cellWidth),
		Y:      int(float64(rect.Y) + float64(row-1)*cellHeight),
		Width:  int(cellWidth),
		Height: int(cellHeight),
	}
}

// ClickGrid 点击网格位置
func ClickGrid(rect auto.Region, gridStr string, opts ...auto.Option) error {
	pos, err := CalculateGridCenterFromString(rect, gridStr)
	if err != nil {
		return fmt.Errorf("计算网格位置失败: %w", err)
	}

	o := auto.ApplyOptions(opts...)
	return input.ClickAt(pos.X+o.ClickOffset.X, pos.Y+o.ClickOffset.Y, o)
}
