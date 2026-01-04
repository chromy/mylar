package utils

import (
	"encoding/binary"
	"github.com/chromy/viz/internal/constants"
	"github.com/go-git/go-git/v5/plumbing"
	"math"
)

func HashToInt32(hash plumbing.Hash) int32 {
	return int32(binary.LittleEndian.Uint32(hash[:]))
}

// LodToSize converts a level of detail to the corresponding size
// lod 0 -> TILE_SIZE
// lod 1 -> TILE_SIZE*2
// lod 2 -> TILE_SIZE*4
// etc
func LodToSize(lod int) int {
	return constants.TileSize * int(math.Pow(2, float64(lod)))
}

// We convert freely between three spaces:
// 'line space' is a 1D space from 0..layout.LastLine
// WorldPosition is a 2D space. We map each LinePosition onto a single
// WorldPosition. World space is a 2D square.
//
// We use a Hilbert Curve for this mapping to preserve locality.
// If line space is 0..n then the world square side length is n = 2^k
// World space is divided into lod=0 tiles of size TILE_SIZE. Each
// (X, Y) in world space can be mapped into a single point into a
// single tile. If (wx, wy) then tx=wx//TILE_SIZE ty=wy//TILE_SIZE
// and the offset within the tile is (wx % TILE_SIZE, wy % TILE_SIZE).

type LinePosition int64

type TileLayout struct {
	LineCount LinePosition
}

// GridSide calculates the side length of the square grid required to
// hold all lines up to LastLine. The side length is always a power of
// 2.
func (l TileLayout) GridSideLength() int64 {
	m := int64(l.LineCount)
	// We need total area >= m+1
	// Indices are [0..LineCount)
	// Side length = 2^ceil(log2(sqrt(m+1)))
	k := math.Ceil(math.Log2(math.Sqrt(float64(m))))
	return int64(math.Pow(2, k))
}

type WorldPosition struct {
	X int64
	Y int64
}

type TilePosition struct {
	Lod     int64
	TileX   int64
	TileY   int64
	OffsetX int64
	OffsetY int64
}

func LineToWorld(line LinePosition, layout TileLayout) WorldPosition {
	n := layout.GridSideLength()

	x, y := hilbertPoint(n, int64(line))
	return WorldPosition{X: x, Y: y}
}

func WorldToTile(world WorldPosition, layout TileLayout) TilePosition {
	tileX := world.X / int64(constants.TileSize)
	tileY := world.Y / int64(constants.TileSize)
	offsetX := world.X % int64(constants.TileSize)
	offsetY := world.Y % int64(constants.TileSize)

	return TilePosition{
		Lod:     0,
		TileX:   tileX,
		TileY:   tileY,
		OffsetX: offsetX,
		OffsetY: offsetY,
	}
}

func TileToWorld(tile TilePosition, layout TileLayout) WorldPosition {
	tileSize := int64(LodToSize(int(tile.Lod)))
	worldX := tile.TileX*tileSize + tile.OffsetX
	worldY := tile.TileY*tileSize + tile.OffsetY

	return WorldPosition{X: worldX, Y: worldY}
}

func WorldToLine(world WorldPosition, layout TileLayout) LinePosition {
	n := layout.GridSideLength()

	d := hilbertIndex(n, world.X, world.Y)
	return LinePosition(d)
}

// rot rotates and flips the quadrant for the Hilbert curve
func rot(n int64, x, y *int64, rx, ry int64) {
	if ry == 0 {
		if rx == 1 {
			*x = n - 1 - *x
			*y = n - 1 - *y
		}
		// Swap x and y
		*x, *y = *y, *x
	}
}

// hilbertPoint maps a 1D distance d to (x,y) coordinates on a grid of size n*n
func hilbertPoint(n int64, d int64) (int64, int64) {
	var rx, ry, s, t, x, y int64
	t = d
	x = 0
	y = 0
	for s = 1; s < n; s *= 2 {
		rx = 1 & (t / 2)
		ry = 1 & (t ^ rx)
		rot(s, &x, &y, rx, ry)
		x += s * rx
		y += s * ry
		t /= 4
	}
	return x, y
}

// hilbertIndex maps (x,y) coordinates to a 1D distance d on a grid of size n*n
func hilbertIndex(n, x, y int64) int64 {
	var rx, ry, s, d int64
	d = 0
	for s = n / 2; s > 0; s /= 2 {
		rx = (x & s) / s // 1 if x has bit s, else 0
		ry = (y & s) / s
		d += s * s * ((3 * rx) ^ ry)
		rot(s, &x, &y, rx, ry)
	}
	return d
}
