import o from "ospec";
import { vec2 } from "gl-matrix";
import { aabb } from "./aabb.js";

o.spec("aabb", () => {
  o.spec("create", () => {
    o("creates zero-initialized AABB", () => {
      const box = aabb.create();
      o(box[0]).equals(0);
      o(box[1]).equals(0);
      o(box[2]).equals(0);
      o(box[3]).equals(0);
    });
  });

  o.spec("fromValues", () => {
    o("creates AABB with specified values", () => {
      const box = aabb.fromValues(1, 2, 3, 4);
      o(box[0]).equals(1);
      o(box[1]).equals(2);
      o(box[2]).equals(3);
      o(box[3]).equals(4);
    });
  });

  o.spec("width and height", () => {
    o("calculates correct dimensions", () => {
      const box = aabb.fromValues(10, 20, 50, 80);
      o(aabb.width(box)).equals(40);
      o(aabb.height(box)).equals(60);
    });
  });

  o.spec("area", () => {
    o("calculates correct area", () => {
      const box = aabb.fromValues(0, 0, 10, 5);
      o(aabb.area(box)).equals(50);
    });
  });

  o.spec("perimeter", () => {
    o("calculates correct perimeter", () => {
      const box = aabb.fromValues(0, 0, 10, 5);
      o(aabb.perimeter(box)).equals(30);
    });
  });

  o.spec("isEmpty", () => {
    o("returns true for empty AABBs", () => {
      const emptyBox1 = aabb.fromValues(10, 0, 5, 10);
      const emptyBox2 = aabb.fromValues(0, 10, 10, 5);
      o(aabb.isEmpty(emptyBox1)).equals(true);
      o(aabb.isEmpty(emptyBox2)).equals(true);
    });

    o("returns false for valid AABBs", () => {
      const validBox = aabb.fromValues(0, 0, 10, 10);
      o(aabb.isEmpty(validBox)).equals(false);
    });
  });

  o.spec("containsPoint", () => {
    o("returns true for points inside AABB", () => {
      const box = aabb.fromValues(0, 0, 10, 10);
      const point = vec2.fromValues(5, 5);
      o(aabb.containsPoint(box, point)).equals(true);
    });

    o("returns true for points on boundary", () => {
      const box = aabb.fromValues(0, 0, 10, 10);
      const corner = vec2.fromValues(0, 0);
      const edge = vec2.fromValues(10, 5);
      o(aabb.containsPoint(box, corner)).equals(true);
      o(aabb.containsPoint(box, edge)).equals(true);
    });

    o("returns false for points outside AABB", () => {
      const box = aabb.fromValues(0, 0, 10, 10);
      const outside = vec2.fromValues(15, 5);
      o(aabb.containsPoint(box, outside)).equals(false);
    });
  });

  o.spec("overlaps", () => {
    o("returns true for overlapping AABBs", () => {
      const box1 = aabb.fromValues(0, 0, 10, 10);
      const box2 = aabb.fromValues(5, 5, 15, 15);
      o(aabb.overlaps(box1, box2)).equals(true);
      o(aabb.overlaps(box2, box1)).equals(true);
    });

    o("returns false for non-overlapping AABBs", () => {
      const box1 = aabb.fromValues(0, 0, 5, 5);
      const box2 = aabb.fromValues(10, 10, 15, 15);
      o(aabb.overlaps(box1, box2)).equals(false);
    });

    o("returns false for touching AABBs", () => {
      const box1 = aabb.fromValues(0, 0, 5, 5);
      const box2 = aabb.fromValues(5, 0, 10, 5);
      o(aabb.overlaps(box1, box2)).equals(false);
    });
  });

  o.spec("union", () => {
    o("creates union of two AABBs", () => {
      const box1 = aabb.fromValues(0, 0, 5, 5);
      const box2 = aabb.fromValues(3, 3, 8, 8);
      const result = aabb.create();
      aabb.union(result, box1, box2);
      
      o(result[0]).equals(0);
      o(result[1]).equals(0);
      o(result[2]).equals(8);
      o(result[3]).equals(8);
    });
  });

  o.spec("translate", () => {
    o("translates AABB by offset", () => {
      const box = aabb.fromValues(0, 0, 10, 10);
      const offset = vec2.fromValues(5, -3);
      const result = aabb.create();
      aabb.translate(result, box, offset);
      
      o(result[0]).equals(5);
      o(result[1]).equals(-3);
      o(result[2]).equals(15);
      o(result[3]).equals(7);
    });
  });

  o.spec("equals", () => {
    o("returns true for equal AABBs within tolerance", () => {
      const box1 = aabb.fromValues(1, 2, 3, 4);
      const box2 = aabb.fromValues(1.0000001, 2.0000001, 3.0000001, 4.0000001);
      o(aabb.equals(box1, box2)).equals(true);
    });

    o("returns false for different AABBs", () => {
      const box1 = aabb.fromValues(1, 2, 3, 4);
      const box2 = aabb.fromValues(1, 2, 3, 5);
      o(aabb.equals(box1, box2)).equals(false);
    });
  });
});