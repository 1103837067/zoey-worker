package auto

import (
	"testing"
)

func TestParseGridPosition(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *GridPosition
		wantErr bool
	}{
		{
			name:  "valid 2x2 grid position 1,1",
			input: "2.2.1.1",
			want:  &GridPosition{Rows: 2, Cols: 2, Row: 1, Col: 1},
		},
		{
			name:  "valid 2x2 grid position 2,2",
			input: "2.2.2.2",
			want:  &GridPosition{Rows: 2, Cols: 2, Row: 2, Col: 2},
		},
		{
			name:  "valid 3x3 grid position 2,2",
			input: "3.3.2.2",
			want:  &GridPosition{Rows: 3, Cols: 3, Row: 2, Col: 2},
		},
		{
			name:  "valid 4x2 grid",
			input: "4.2.3.1",
			want:  &GridPosition{Rows: 4, Cols: 2, Row: 3, Col: 1},
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid format - too few parts",
			input:   "2.2.1",
			wantErr: true,
		},
		{
			name:    "invalid format - too many parts",
			input:   "2.2.1.1.1",
			wantErr: true,
		},
		{
			name:    "invalid rows - not a number",
			input:   "a.2.1.1",
			wantErr: true,
		},
		{
			name:    "invalid - rows < 1",
			input:   "0.2.1.1",
			wantErr: true,
		},
		{
			name:    "invalid - row > rows",
			input:   "2.2.3.1",
			wantErr: true,
		},
		{
			name:    "invalid - col > cols",
			input:   "2.2.1.3",
			wantErr: true,
		},
		{
			name:    "invalid - row < 1",
			input:   "2.2.0.1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseGridPosition(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseGridPosition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Rows != tt.want.Rows || got.Cols != tt.want.Cols ||
					got.Row != tt.want.Row || got.Col != tt.want.Col {
					t.Errorf("ParseGridPosition() = %+v, want %+v", got, tt.want)
				}
			}
		})
	}
}

func TestFormatGridPosition(t *testing.T) {
	result := FormatGridPosition(2, 2, 1, 1)
	if result != "2.2.1.1" {
		t.Errorf("FormatGridPosition() = %v, want %v", result, "2.2.1.1")
	}

	result = FormatGridPosition(3, 4, 2, 3)
	if result != "3.4.2.3" {
		t.Errorf("FormatGridPosition() = %v, want %v", result, "3.4.2.3")
	}
}

func TestCalculateGridCenter(t *testing.T) {
	rect := Region{X: 100, Y: 100, Width: 200, Height: 200}

	tests := []struct {
		name string
		grid *GridPosition
		want Point
	}{
		{
			name: "2x2 grid - top left (1,1)",
			grid: &GridPosition{Rows: 2, Cols: 2, Row: 1, Col: 1},
			want: Point{X: 150, Y: 150}, // 100 + 0.5*100 = 150
		},
		{
			name: "2x2 grid - top right (1,2)",
			grid: &GridPosition{Rows: 2, Cols: 2, Row: 1, Col: 2},
			want: Point{X: 250, Y: 150}, // 100 + 1.5*100 = 250
		},
		{
			name: "2x2 grid - bottom left (2,1)",
			grid: &GridPosition{Rows: 2, Cols: 2, Row: 2, Col: 1},
			want: Point{X: 150, Y: 250},
		},
		{
			name: "2x2 grid - bottom right (2,2)",
			grid: &GridPosition{Rows: 2, Cols: 2, Row: 2, Col: 2},
			want: Point{X: 250, Y: 250},
		},
		{
			name: "nil grid - center of rect",
			grid: nil,
			want: Point{X: 200, Y: 200}, // 100 + 200/2 = 200
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateGridCenter(rect, tt.grid)
			if got.X != tt.want.X || got.Y != tt.want.Y {
				t.Errorf("CalculateGridCenter() = (%d, %d), want (%d, %d)",
					got.X, got.Y, tt.want.X, tt.want.Y)
			}
		})
	}
}

func TestCalculateGridCenter_3x3(t *testing.T) {
	rect := Region{X: 0, Y: 0, Width: 300, Height: 300}

	// 3x3 grid, each cell is 100x100
	// Cell (2,2) center should be at (150, 150)
	grid := &GridPosition{Rows: 3, Cols: 3, Row: 2, Col: 2}
	got := CalculateGridCenter(rect, grid)

	// Cell width = 300/3 = 100
	// Center of (2,2): x = 0 + (2-0.5)*100 = 150, y = 0 + (2-0.5)*100 = 150
	if got.X != 150 || got.Y != 150 {
		t.Errorf("CalculateGridCenter(3x3, 2,2) = (%d, %d), want (150, 150)",
			got.X, got.Y)
	}
}

func TestCalculateGridCenterFromString(t *testing.T) {
	rect := Region{X: 100, Y: 100, Width: 200, Height: 200}

	// Valid grid string
	pos, err := CalculateGridCenterFromString(rect, "2.2.1.1")
	if err != nil {
		t.Errorf("CalculateGridCenterFromString() error = %v", err)
	}
	if pos.X != 150 || pos.Y != 150 {
		t.Errorf("CalculateGridCenterFromString() = (%d, %d), want (150, 150)",
			pos.X, pos.Y)
	}

	// Empty string - should return center
	pos, err = CalculateGridCenterFromString(rect, "")
	if err != nil {
		t.Errorf("CalculateGridCenterFromString(\"\") error = %v", err)
	}
	if pos.X != 200 || pos.Y != 200 {
		t.Errorf("CalculateGridCenterFromString(\"\") = (%d, %d), want (200, 200)",
			pos.X, pos.Y)
	}

	// Invalid string
	_, err = CalculateGridCenterFromString(rect, "invalid")
	if err == nil {
		t.Error("CalculateGridCenterFromString(invalid) should return error")
	}
}

func TestGetGridCellRect(t *testing.T) {
	rect := Region{X: 100, Y: 100, Width: 200, Height: 200}

	// 2x2 grid, cell (1,1)
	cell := GetGridCellRect(rect, 2, 2, 1, 1)
	if cell.X != 100 || cell.Y != 100 || cell.Width != 100 || cell.Height != 100 {
		t.Errorf("GetGridCellRect(2,2,1,1) = %+v, want {100,100,100,100}", cell)
	}

	// 2x2 grid, cell (2,2)
	cell = GetGridCellRect(rect, 2, 2, 2, 2)
	if cell.X != 200 || cell.Y != 200 || cell.Width != 100 || cell.Height != 100 {
		t.Errorf("GetGridCellRect(2,2,2,2) = %+v, want {200,200,100,100}", cell)
	}

	// 2x2 grid, cell (1,2)
	cell = GetGridCellRect(rect, 2, 2, 1, 2)
	if cell.X != 200 || cell.Y != 100 || cell.Width != 100 || cell.Height != 100 {
		t.Errorf("GetGridCellRect(2,2,1,2) = %+v, want {200,100,100,100}", cell)
	}
}

func TestGridIterator(t *testing.T) {
	rect := Region{X: 0, Y: 0, Width: 200, Height: 200}
	iter := NewGridIterator(rect, 2, 2)

	if iter.Count() != 4 {
		t.Errorf("GridIterator.Count() = %d, want 4", iter.Count())
	}

	// Iterate through all positions
	var positions []Point
	for {
		pos := iter.Next()
		if pos == nil {
			break
		}
		positions = append(positions, *pos)
	}

	if len(positions) != 4 {
		t.Errorf("GridIterator returned %d positions, want 4", len(positions))
	}

	// Expected positions for 2x2 grid on 200x200 rect
	// Cell size: 100x100
	// (1,1): center at (50, 50)
	// (1,2): center at (150, 50)
	// (2,1): center at (50, 150)
	// (2,2): center at (150, 150)
	expected := []Point{
		{X: 50, Y: 50},
		{X: 150, Y: 50},
		{X: 50, Y: 150},
		{X: 150, Y: 150},
	}

	for i, pos := range positions {
		if pos.X != expected[i].X || pos.Y != expected[i].Y {
			t.Errorf("Position %d: got (%d, %d), want (%d, %d)",
				i, pos.X, pos.Y, expected[i].X, expected[i].Y)
		}
	}

	// Test reset
	iter.Reset()
	pos := iter.Next()
	if pos == nil || pos.X != 50 || pos.Y != 50 {
		t.Error("GridIterator.Reset() did not reset correctly")
	}
}

func TestGridIterator_3x2(t *testing.T) {
	rect := Region{X: 0, Y: 0, Width: 300, Height: 200}
	iter := NewGridIterator(rect, 2, 3) // 2 rows, 3 cols

	if iter.Count() != 6 {
		t.Errorf("GridIterator.Count() = %d, want 6", iter.Count())
	}

	var count int
	for iter.Next() != nil {
		count++
	}

	if count != 6 {
		t.Errorf("GridIterator iterated %d times, want 6", count)
	}
}

// BenchmarkParseGridPosition 基准测试
func BenchmarkParseGridPosition(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseGridPosition("3.3.2.2")
	}
}

// BenchmarkCalculateGridCenter 基准测试
func BenchmarkCalculateGridCenter(b *testing.B) {
	rect := Region{X: 100, Y: 100, Width: 200, Height: 200}
	grid := &GridPosition{Rows: 3, Cols: 3, Row: 2, Col: 2}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateGridCenter(rect, grid)
	}
}
