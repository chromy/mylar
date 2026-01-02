package utils

import (
	"github.com/chromy/viz/internal/constants"
	"github.com/go-git/go-git/v5/plumbing"
	"testing"
)

func TestHashToInt53(t *testing.T) {
	tests := []struct {
		hashAsString string
		want         int64
	}{
		{"", 0},
		{"09030652af16811842314a2c8fa5e344c2bb5c34", 306417227924233},
		{"c5ff5b84be06c42e15a35a312a7a2bb3760d29d9", 1133315241017285},
	}

	const maxSafeInteger int64 = 9007199254740991
	const minSafeInteger int64 = -9007199254740991

	for _, tt := range tests {
		hash := plumbing.NewHash(tt.hashAsString)
		got := HashToInt53(hash)
		if got != tt.want {
			t.Errorf("HashToInt53(%s) = %d, want %d", hash, got, tt.want)
		}
		if got > maxSafeInteger {
			t.Errorf("HashToInt53(%s) = %d, larger than Number.MAX_SAFE_INTEGER (%d)", hash, got, maxSafeInteger)
		}
		if got < minSafeInteger {
			t.Errorf("HashToInt53(%s) = %d, smaller than Number.MIN_SAFE_INTEGER (%d)", hash, got, minSafeInteger)
		}
	}
}

func TestHashToInt32(t *testing.T) {
	tests := []struct {
		hashAsString string
		want         int32
	}{
		{"", 0},
		{"09030652af16811842314a2c8fa5e344c2bb5c34", 1376125705},
		{"c5ff5b84be06c42e15a35a312a7a2bb3760d29d9", -2074345531},
	}

	for _, tt := range tests {
		hash := plumbing.NewHash(tt.hashAsString)
		got := HashToInt32(hash)
		if got != tt.want {
			t.Errorf("HashToInt32(%s) = %d, want %d", hash, got, tt.want)
		}
	}
}

func TestMortonEncoding(t *testing.T) {
	tests := []struct {
		x, y uint32
		want uint64
	}{
		{0, 0, 0},
		{1, 0, 1},
		{0, 1, 2},
		{1, 1, 3},
		{2, 0, 4},
		{0, 2, 8},
		{2, 2, 12},
		{255, 255, 0xFFFF},
	}

	for _, tt := range tests {
		got := mortonEncode(tt.x, tt.y)
		if got != tt.want {
			t.Errorf("mortonEncode(%d, %d) = %d, want %d", tt.x, tt.y, got, tt.want)
		}

		// Test round-trip
		x, y := mortonDecode(got)
		if x != tt.x || y != tt.y {
			t.Errorf("mortonDecode(%d) = (%d, %d), want (%d, %d)", got, x, y, tt.x, tt.y)
		}
	}
}

func TestLineToWorldAndBack(t *testing.T) {
	layout := TileLayout{LastLine: 1023} // 32x32 grid

	tests := []LinePosition{0, 1, 2, 3, 4, 8, 12, 100, 500, 1023}

	for _, line := range tests {
		world := LineToWorld(line, layout)
		backToLine := WorldToLine(world, layout)

		if backToLine != line {
			t.Errorf("Round-trip failed for line %d: got %d", line, backToLine)
		}
	}
}

func TestWorldToTileAndBack(t *testing.T) {
	layout := TileLayout{LastLine: 1023}

	tests := []WorldPosition{
		{0, 0},
		{1, 1},
		{63, 63},
		{64, 64},
		{65, 65},
		{128, 192},
		{255, 255},
	}

	for _, world := range tests {
		tile := WorldToTile(world, layout)
		backToWorld := TileToWorld(tile, layout)

		if backToWorld != world {
			t.Errorf("Round-trip failed for world %+v: got %+v", world, backToWorld)
		}
	}
}

func TestWorldToTileProperties(t *testing.T) {
	layout := TileLayout{LastLine: 1023}

	// Test that coordinates within same tile map to same tile position
	world1 := WorldPosition{10, 20}
	world2 := WorldPosition{30, 40}

	tile1 := WorldToTile(world1, layout)
	tile2 := WorldToTile(world2, layout)

	if tile1.TileX != tile2.TileX || tile1.TileY != tile2.TileY {
		t.Errorf("Expected same tile for positions within same tile")
	}

	// Test tile boundary
	worldOnBoundary := WorldPosition{64, 64}
	tile := WorldToTile(worldOnBoundary, layout)

	expectedTileX := int64(64 / constants.TileSize)
	expectedTileY := int64(64 / constants.TileSize)

	if tile.TileX != expectedTileX || tile.TileY != expectedTileY {
		t.Errorf("Tile position incorrect at boundary: got (%d,%d), want (%d,%d)",
			tile.TileX, tile.TileY, expectedTileX, expectedTileY)
	}
}

func TestTileToWorldWithLod(t *testing.T) {
	layout := TileLayout{LastLine: 1023}

	// Test different LOD levels
	tests := []struct {
		tile TilePosition
		want WorldPosition
	}{
		{TilePosition{Lod: 0, TileX: 1, TileY: 1, OffsetX: 10, OffsetY: 20}, WorldPosition{74, 84}},
		{TilePosition{Lod: 1, TileX: 1, TileY: 1, OffsetX: 10, OffsetY: 20}, WorldPosition{138, 148}},
		{TilePosition{Lod: 2, TileX: 1, TileY: 1, OffsetX: 10, OffsetY: 20}, WorldPosition{266, 276}},
	}

	for _, tt := range tests {
		got := TileToWorld(tt.tile, layout)
		if got != tt.want {
			t.Errorf("TileToWorld(%+v) = %+v, want %+v", tt.tile, got, tt.want)
		}
	}
}

func TestInitialSize(t *testing.T) {
	tests := []struct {
		m    int
		want int
	}{
		{1, 2},
		{4, 2},
		{5, 4},
		{16, 4},
		{17, 8},
		{64, 8},
		{65, 16},
		{256, 16},
		{257, 32},
	}

	for _, tt := range tests {
		got := InitialSize(tt.m)
		if got != tt.want {
			t.Errorf("InitialSize(%d) = %d, want %d", tt.m, got, tt.want)
		}
	}
}

func TestLodToSize(t *testing.T) {
	tests := []struct {
		lod  int
		want int
	}{
		{0, 64},
		{1, 128},
		{2, 256},
		{3, 512},
	}

	for _, tt := range tests {
		got := LodToSize(tt.lod)
		if got != tt.want {
			t.Errorf("LodToSize(%d) = %d, want %d", tt.lod, got, tt.want)
		}
	}
}

func TestCoordinateSystemConsistency(t *testing.T) {
	layout := TileLayout{LastLine: 255} // Small grid for testing

	// Test that all transformations are consistent
	for line := LinePosition(0); line <= 255; line++ {
		// Line -> World -> Line
		world := LineToWorld(line, layout)
		backToLine := WorldToLine(world, layout)
		if backToLine != line {
			t.Errorf("Line->World->Line inconsistent for line %d", line)
		}

		// World -> Tile -> World (should be consistent)
		tile := WorldToTile(world, layout)
		backToWorld := TileToWorld(tile, layout)
		if backToWorld != world {
			t.Errorf("World->Tile->World inconsistent for world %+v", world)
		}
	}
}
