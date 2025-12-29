import { aabb } from "./aabb.tsx";

export function nextPowerOfTwo(n) {
  if (n <= 0) return 1;
  if ((n & (n - 1)) === 0) return n * 2;
  
  let power = 1;
  while (power <= n) {
    power *= 2;
  }
  return power;
}

export function* quadtreeAABBs(m) {
  if (m <= 0) return;
  
  const k = Math.ceil(Math.log2(Math.sqrt(m)));
  const initialSize = 2 ** k;
  
  const queue = [aabb.fromValues(0, 0, initialSize, initialSize)];
  
  while (queue.length > 0) {
    const currentAABB = queue.shift();
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
        aabb.fromValues(minX + halfSize, minY + halfSize, minX + size, minY + size)
      );
    }
  }
}