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

export function* quadtreeAABBs(m: number): Generator<aabb> {
  if (m <= 0) {
    return;
  }

  const k = Math.ceil(Math.log2(Math.sqrt(m)));
  const initialSize = 2 ** k;

  const queue = [aabb.fromValues(0, 0, initialSize, initialSize)];

  while (queue.length > 0) {
    const currentAABB = queue.shift();
    if (!currentAABB) break;
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

export function toLod(box: aabb): number {
  // width === TILE_SIZE -> 0
  // width === TILE_SIZE*2 -> 1
  // width === TILE_SIZE*4 -> 2
  // etc
  return Math.log2(aabb.width(box) / TILE_SIZE);
}

export function* requiredTiles(
  bounds: aabb,
  lineCount: number,
): Generator<aabb> {
  const limit = 100;
  let count = 0;

  for (const quadAABB of quadtreeAABBs(lineCount)) {
    if (count >= limit) {
      break;
    }
    if (toLod(quadAABB) < 0) {
      break;
    }
    if (aabb.overlaps(quadAABB, bounds)) {
      yield quadAABB;
      count += 1;
    }
  }
}
