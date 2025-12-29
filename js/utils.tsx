import { TILE_SIZE } from "./schemas.js";

// lod 0 -> TILE_SIZE
// lod 1 -> TILE_SIZE*2
// lod 2 -> TILE_SIZE*4
// etc
export function lodToSize(lod: number): number {
  return TILE_SIZE * Math.pow(2, lod);
}

export function initialSize(m: number): number {
  const k = Math.ceil(Math.log2(Math.sqrt(m)));
  const initialSize = 2 ** k;
  return initialSize;
}
