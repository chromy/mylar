package utils

import (
	"github.com/chromy/viz/internal/constants"
	"math"
)

// LodToSize converts a level of detail to the corresponding size
// lod 0 -> TILE_SIZE
// lod 1 -> TILE_SIZE*2
// lod 2 -> TILE_SIZE*4
// etc
func LodToSize(lod int) int {
	return constants.TileSize * int(math.Pow(2, float64(lod)))
}

func InitialSize(m int) int {
	if m <= 1 {
		return 2
	}
	k := math.Ceil(math.Log2(math.Sqrt(float64(m))))
	return int(math.Pow(2, k))
}

// We convert freely between three spaces:
// 'line space' is a 1D space from 0..layout.LastLine
// WorldPosition is a 2D space. We map each LinePosition onto a single
// WorldPosition. World space is a 2D square. If line space is 0..n
// then the world square is (0..2**k), (0..2**k) st. k is the smallest
// integer n <= 2**k * 2**k
// World space is divided into lod=0 tiles of size TILE_SIZE. Each
// (X, Y) in world space can be mapped into a single point into a
// single  tile. If (wx, wy) then tx=wx//TILE_SIZE ty=wy//TILE_SIZE
// and the offset within the tile is (wx % TILE_SIZE, wy % TILE_SIZE).

type TileLayout struct {
	LastLine LinePosition
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

type LinePosition int64

func LineToWorld(line LinePosition, layout TileLayout) WorldPosition {
	x, y := mortonDecode(uint64(line))
	return WorldPosition{X: int64(x), Y: int64(y)}
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
	encoded := mortonEncode(uint32(world.X), uint32(world.Y))
	return LinePosition(encoded)
}

// mortonEncode interleaves the bits of x and y to produce a Morton code
func mortonEncode(x, y uint32) uint64 {
	return uint64(spreadBits(x)) | (uint64(spreadBits(y)) << 1)
}

// mortonDecode extracts x and y from a Morton code
func mortonDecode(code uint64) (uint32, uint32) {
	x := compactBits(uint32(code))
	y := compactBits(uint32(code >> 1))
	return x, y
}

// spreadBits spreads the bits of a 16-bit number across 32 bits
func spreadBits(x uint32) uint32 {
	x = (x | (x << 8)) & 0x00FF00FF
	x = (x | (x << 4)) & 0x0F0F0F0F
	x = (x | (x << 2)) & 0x33333333
	x = (x | (x << 1)) & 0x55555555
	return x
}

// compactBits compacts spread bits back to a 16-bit number
func compactBits(x uint32) uint32 {
	x = x & 0x55555555
	x = (x ^ (x >> 1)) & 0x33333333
	x = (x ^ (x >> 2)) & 0x0F0F0F0F
	x = (x ^ (x >> 4)) & 0x00FF00FF
	x = (x ^ (x >> 8)) & 0x0000FFFF
	return x
}
