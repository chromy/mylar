import o from "ospec";
import { nextPowerOfTwo, toLod, lodToSize, quadtreeBoundingBox } from "./math.js";
import { aabb } from "./aabb.js";

o.spec("math", () => {
  o.spec("nextPowerOfTwo", () => {
    o("returns 1 for non-positive numbers", () => {
      o(nextPowerOfTwo(-1)).equals(1);
      o(nextPowerOfTwo(0)).equals(1);
    });

    o("doubles power of two numbers", () => {
      o(nextPowerOfTwo(1)).equals(2);
      o(nextPowerOfTwo(2)).equals(4);
      o(nextPowerOfTwo(4)).equals(8);
      o(nextPowerOfTwo(16)).equals(32);
    });

    o("finds next power of two for non-power-of-two numbers", () => {
      o(nextPowerOfTwo(3)).equals(4);
      o(nextPowerOfTwo(5)).equals(8);
      o(nextPowerOfTwo(10)).equals(16);
      o(nextPowerOfTwo(100)).equals(128);
    });
  });

  o.spec("toLod", () => {
    o("calculates correct LOD for different box widths", () => {
      const box64 = aabb.fromValues(0, 0, 64, 64);
      o(toLod(box64)).equals(0);

      const box128 = aabb.fromValues(0, 0, 128, 128);
      o(toLod(box128)).equals(1);

      const box256 = aabb.fromValues(0, 0, 256, 256);
      o(toLod(box256)).equals(2);
    });
  });

  o.spec("lodToSize", () => {
    o("converts LOD to correct size", () => {
      o(lodToSize(0)).equals(64);
      o(lodToSize(1)).equals(128);
      o(lodToSize(2)).equals(256);
      o(lodToSize(3)).equals(512);
    });
  });

  o.spec("quadtreeBoundingBox", () => {
    o("creates correct bounding box for different values", () => {
      const box1 = quadtreeBoundingBox(1);
      o(aabb.width(box1)).equals(1);
      o(aabb.height(box1)).equals(1);

      const box5 = quadtreeBoundingBox(5);
      o(aabb.width(box5)).equals(4);
      o(aabb.height(box5)).equals(4);

      const box100 = quadtreeBoundingBox(100);
      o(aabb.width(box100)).equals(16);
      o(aabb.height(box100)).equals(16);
    });

    o("always starts at origin", () => {
      const box = quadtreeBoundingBox(42);
      o(box[0]).equals(0);
      o(box[1]).equals(0);
    });
  });
});