import { aabb } from "./aabb.js";
import { TILE_SIZE } from "./schemas.js";

export function nextPowerOfTwo(n: number): number {
  if (n <= 0) {
    return 1;
  }
  if ((n & (n - 1)) === 0) {
    return n * 2;
  }

  let power = 1;
  while (power <= n) {
    power *= 2;
  }
  return power;
}

export function toLod(box: aabb): number {
  return Math.log2(aabb.width(box) / TILE_SIZE);
}

// lod 0 -> TILE_SIZE
// lod 1 -> TILE_SIZE*2
// lod 2 -> TILE_SIZE*4
// etc
export function lodToSize(lod: number): number {
  return TILE_SIZE * Math.pow(2, lod);
}


export function quadtreeBoundingBox(m: number): aabb {
  const k = Math.ceil(Math.log2(Math.sqrt(m)));
  const initialSize = 2 ** k;
  return aabb.fromValues(0, 0, initialSize, initialSize);
}

export function* quadtreeAABBs(m: number, predicate: (box: aabb) => boolean): Generator<aabb> {
  if (m <= 0) {
    return;
  }

  const queue = [quadtreeBoundingBox(m)];

  while (queue.length > 0) {
    const currentAABB = queue.shift();
    if (!currentAABB) {
      break;
    }

    if (predicate(currentAABB)) {
      yield currentAABB;

      const size = aabb.width(currentAABB);
      if (size > 1) {
        const halfSize = size / 2;
        const minX = currentAABB[0];
        const minY = currentAABB[1];

        queue.push(
          aabb.fromValues(minX, minY, minX + halfSize, minY + halfSize),
          aabb.fromValues(minX + halfSize, minY, minX + size, minY + halfSize),
          aabb.fromValues(minX, minY + halfSize, minX + halfSize, minY + size),
          aabb.fromValues(
            minX + halfSize,
            minY + halfSize,
            minX + size,
            minY + size,
          ),
        );
      }
    }
  }
}

export function* requiredTiles(
  bounds: aabb,
  lineCount: number,
): Generator<aabb> {
  const limit = 100;
  let count = 0;

  const inScope = (box: aabb) => aabb.overlaps(box, bounds);

  for (const quadAABB of quadtreeAABBs(lineCount, inScope)) {
    if (count >= limit) {
      break;
    }
    if (toLod(quadAABB) < 0) {
      break;
    }
    yield quadAABB;
    count += 1;
  }
}
