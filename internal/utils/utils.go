package utils

import (
	"math"
	"github.com/chromy/viz/internal/constants"
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
	Lod int64
	TileX int64
	TileY int64
	OffsetX int64
	OffsetY int64
}

type LinePosition int64

func LineToWorld(line LinePosition, layout TileLayout) WorldPosition {
}

func WorldToTile(world WorldPosition, layout TileLayout) TilePosition {
}

func TileToWorld(tile TilePosition, layout TileLayout) WorldPosition {
}

func WorldToLine(world WorldPosition, layout TileLayout) LinePosition {
}
