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

type TileLayout struct {
  lineCount: number;
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


