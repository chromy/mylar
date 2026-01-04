import { TILE_SIZE } from "./schemas.js";

// lod 0 -> TILE_SIZE
// lod 1 -> TILE_SIZE*2
// lod 2 -> TILE_SIZE*4
// etc
export function lodToSize(lod: number): number {
  return TILE_SIZE * Math.pow(2, lod);
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

export type LinePosition = number;

export interface TileLayout {
  lineCount: LinePosition;
}

export interface WorldPosition {
  x: number;
  y: number;
}

export interface TilePosition {
  lod: number;
  tileX: number;
  tileY: number;
  offsetX: number;
  offsetY: number;
}

// Helper to calculate the side length of the square grid (N).
// N must be a power of 2.
export function getGridSide(layout: TileLayout): number {
  const m = layout.lineCount;
  // Side length = 2^ceil(log2(sqrt(m)))
  const k = Math.ceil(Math.log2(Math.sqrt(m)));
  return Math.pow(2, k);
}

export function lineToWorld(
  line: LinePosition,
  layout: TileLayout,
): WorldPosition {
  const n = getGridSide(layout);
  const [x, y] = hilbertPoint(n, line);

  return { x, y };
}

export function worldToTile(
  world: WorldPosition,
  layout: TileLayout,
): TilePosition {
  const tileX = Math.floor(world.x / TILE_SIZE);
  const tileY = Math.floor(world.y / TILE_SIZE);
  const offsetX = world.x % TILE_SIZE;
  const offsetY = world.y % TILE_SIZE;

  return {
    lod: 0,
    tileX: tileX,
    tileY: tileY,
    offsetX: offsetX,
    offsetY: offsetY,
  };
}

export function tileToWorld(
  tile: TilePosition,
  layout: TileLayout,
): WorldPosition {
  const tileSize = lodToSize(tile.lod);
  const worldX = tile.tileX * tileSize + tile.offsetX;
  const worldY = tile.tileY * tileSize + tile.offsetY;

  return { x: worldX, y: worldY };
}

export function worldToLine(
  world: WorldPosition,
  layout: TileLayout,
): LinePosition {
  const n = getGridSide(layout);
  const d = hilbertIndex(n, world.x, world.y);

  return d;
}

/**
 * Rotates and flips the quadrant for the Hilbert curve.
 * Uses standard JS numbers.
 */
function rot(n: number, ref: { x: number; y: number }, rx: number, ry: number) {
  if (ry === 0) {
    if (rx === 1) {
      ref.x = n - 1 - ref.x;
      ref.y = n - 1 - ref.y;
    }
    // Swap x and y
    const temp = ref.x;
    ref.x = ref.y;
    ref.y = temp;
  }
}

/**
 * Maps a 1D distance d to (x,y) coordinates on a grid of size n*n.
 */
function hilbertPoint(n: number, d: number): [number, number] {
  let rx: number,
    ry: number,
    s: number,
    t: number = d;
  const pt = { x: 0, y: 0 };

  for (s = 1; s < n; s *= 2) {
    // 1 & (t / 2)
    rx = 1 & (t >>> 1);
    ry = 1 & (t ^ rx);

    rot(s, pt, rx, ry);

    pt.x += s * rx;
    pt.y += s * ry;

    // t /= 4
    t >>>= 2;
  }
  return [pt.x, pt.y];
}

/**
 * Maps (x,y) coordinates to a 1D distance d on a grid of size n*n.
 */
function hilbertIndex(n: number, x: number, y: number): number {
  let rx: number,
    ry: number,
    s: number,
    d: number = 0;
  // We copy x and y into an object so we can mutate them in rot()
  const pt = { x: x, y: y };

  for (s = n >>> 1; s > 0; s >>>= 1) {
    // Check if the bits corresponding to s are set
    rx = (pt.x & s) > 0 ? 1 : 0;
    ry = (pt.y & s) > 0 ? 1 : 0;

    d += s * s * ((3 * rx) ^ ry);

    rot(s, pt, rx, ry);
  }
  return d;
}
