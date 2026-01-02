import { aabb } from "./aabb.js";
import { TILE_SIZE } from "./schemas.js";
import { initialSize } from "./utils.js";

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

export function quadtreeBoundingBox(m: number): aabb {
  const size = initialSize(m);
  return aabb.fromValues(0, 0, size, size);
}

export function* quadtreeAABBs(
  m: number,
  predicate: (box: aabb) => boolean,
): Generator<aabb> {
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

const MIN_PIXELS_TO_DISPLAY = 256;

export function* requiredTiles(
  bounds: aabb,
  lineCount: number,
  pixelsPerWorldUnit: number,
): Generator<aabb> {
  const inScope = (box: aabb) => aabb.overlaps(box, bounds);
  let count = 0;

  for (const quadAABB of quadtreeAABBs(lineCount, inScope)) {
    const width = aabb.width(quadAABB);
    if (count > 0 && width * pixelsPerWorldUnit < MIN_PIXELS_TO_DISPLAY) {
      break;
    }
    if (toLod(quadAABB) < 0) {
      break;
    }
    count += 1;
    yield quadAABB;
  }
}
