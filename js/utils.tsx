import { TILE_SIZE } from "./schemas.js";

// lod 0 -> TILE_SIZE
// lod 1 -> TILE_SIZE*2
// lod 2 -> TILE_SIZE*4
// etc
export function lodToSize(lod: number): number {
  return TILE_SIZE * Math.pow(2, lod);
}

export function initialSize(m: number): number {
  if (m <= 1) {
    return 2;
  }
  const k = Math.ceil(Math.log2(Math.sqrt(m)));
  return Math.pow(2, k);
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

export interface TileLayout {
  LastLine: LinePosition;
}

export interface WorldPosition {
  X: number;
  Y: number;
}

export interface TilePosition {
  Lod: number;
  TileX: number;
  TileY: number;
  OffsetX: number;
  OffsetY: number;
}

export type LinePosition = number;

export function lineToWorld(line: LinePosition, layout: TileLayout): WorldPosition {
  const [x, y] = mortonDecode(line);
  return { X: x, Y: y };
}

export function worldToTile(world: WorldPosition, layout: TileLayout): TilePosition {
  const tileX = Math.floor(world.X / TILE_SIZE);
  const tileY = Math.floor(world.Y / TILE_SIZE);
  const offsetX = world.X % TILE_SIZE;
  const offsetY = world.Y % TILE_SIZE;

  return {
    Lod: 0,
    TileX: tileX,
    TileY: tileY,
    OffsetX: offsetX,
    OffsetY: offsetY,
  };
}

export function tileToWorld(tile: TilePosition, layout: TileLayout): WorldPosition {
  const tileSize = lodToSize(tile.Lod);
  const worldX = tile.TileX * tileSize + tile.OffsetX;
  const worldY = tile.TileY * tileSize + tile.OffsetY;

  return { X: worldX, Y: worldY };
}

export function worldToLine(world: WorldPosition, layout: TileLayout): LinePosition {
  const encoded = mortonEncode(world.X, world.Y);
  return encoded;
}

// mortonEncode interleaves the bits of x and y to produce a Morton code
export function mortonEncode(x: number, y: number): number {
  return spreadBits(x) | (spreadBits(y) << 1);
}

// mortonDecode extracts x and y from a Morton code
export function mortonDecode(code: number): [number, number] {
  const x = compactBits(code);
  const y = compactBits(code >> 1);
  return [x, y];
}

// spreadBits spreads the bits of a 16-bit number across 32 bits
export function spreadBits(x: number): number {
  x = (x | (x << 8)) & 0x00FF00FF;
  x = (x | (x << 4)) & 0x0F0F0F0F;
  x = (x | (x << 2)) & 0x33333333;
  x = (x | (x << 1)) & 0x55555555;
  return x >>> 0; // Convert to unsigned 32-bit
}

// compactBits compacts spread bits back to a 16-bit number
export function compactBits(x: number): number {
  x = x & 0x55555555;
  x = (x ^ (x >> 1)) & 0x33333333;
  x = (x ^ (x >> 2)) & 0x0F0F0F0F;
  x = (x ^ (x >> 4)) & 0x00FF00FF;
  x = (x ^ (x >> 8)) & 0x0000FFFF;
  return x >>> 0; // Convert to unsigned 32-bit
}
