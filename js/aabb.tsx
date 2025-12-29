import { glMatrix, vec4, vec2 } from "gl-matrix";

export type aabb = vec4;

function create(): aabb {
  const out = new glMatrix.ARRAY_TYPE(4);
  out[0] = 0; // minX
  out[1] = 0; // minY
  out[2] = 0; // maxX
  out[3] = 0; // maxY
  return out;
}

function clone(a: Readonly<aabb>): aabb {
  const out = new glMatrix.ARRAY_TYPE(4);
  out[0] = a[0];
  out[1] = a[1];
  out[2] = a[2];
  out[3] = a[3];
  return out;
}

function copy(out: aabb, a: Readonly<aabb>): aabb {
  out[0] = a[0];
  out[1] = a[1];
  out[2] = a[2];
  out[3] = a[3];
  return out;
}

function fromValues(minX: number, minY: number, maxX: number, maxY: number): aabb {
  const out = new Float32Array(4) as aabb;
  out[0] = minX;
  out[1] = minY;
  out[2] = maxX;
  out[3] = maxY;
  return out;
}

function fromMinMax(out: aabb, min: Readonly<vec2>, max: Readonly<vec2>): aabb {
  out[0] = min[0];
  out[1] = min[1];
  out[2] = max[0];
  out[3] = max[1];
  return out;
}

function fromCenterSize(out: aabb, center: Readonly<vec2>, size: Readonly<vec2>): aabb {
  const halfWidth = size[0] * 0.5;
  const halfHeight = size[1] * 0.5;
  out[0] = center[0] - halfWidth;
  out[1] = center[1] - halfHeight;
  out[2] = center[0] + halfWidth;
  out[3] = center[1] + halfHeight;
  return out;
}

function fromPoint(out: aabb, point: Readonly<vec2>): aabb {
  out[0] = point[0];
  out[1] = point[1];
  out[2] = point[0];
  out[3] = point[1];
  return out;
}

function getMin(out: vec2, a: Readonly<aabb>): vec2 {
  out[0] = a[0];
  out[1] = a[1];
  return out;
}

function getMax(out: vec2, a: Readonly<aabb>): vec2 {
  out[0] = a[2];
  out[1] = a[3];
  return out;
}

function getCenter(out: vec2, a: Readonly<aabb>): vec2 {
  out[0] = (a[0] + a[2]) * 0.5;
  out[1] = (a[1] + a[3]) * 0.5;
  return out;
}

function getSize(out: vec2, a: Readonly<aabb>): vec2 {
  out[0] = a[2] - a[0];
  out[1] = a[3] - a[1];
  return out;
}

function width(a: Readonly<aabb>): number {
  return a[2] - a[0];
}

function height(a: Readonly<aabb>): number {
  return a[3] - a[1];
}

function area(a: Readonly<aabb>): number {
  return (a[2] - a[0]) * (a[3] - a[1]);
}

function perimeter(a: Readonly<aabb>): number {
  const w = a[2] - a[0];
  const h = a[3] - a[1];
  return 2 * (w + h);
}

function isEmpty(a: Readonly<aabb>): boolean {
  return a[0] >= a[2] || a[1] >= a[3];
}

function isValid(a: Readonly<aabb>): boolean {
  return a[0] <= a[2] && a[1] <= a[3];
}

function containsPoint(a: Readonly<aabb>, point: Readonly<vec2>): boolean {
  return point[0] >= a[0] && point[0] <= a[2] && point[1] >= a[1] && point[1] <= a[3];
}

function containsAABB(a: Readonly<aabb>, b: Readonly<aabb>): boolean {
  return b[0] >= a[0] && b[1] >= a[1] && b[2] <= a[2] && b[3] <= a[3];
}

function overlaps(a: Readonly<aabb>, b: Readonly<aabb>): boolean {
  return a[0] < b[2] && a[2] > b[0] && a[1] < b[3] && a[3] > b[1];
}

function union(out: aabb, a: Readonly<aabb>, b: Readonly<aabb>): aabb {
  out[0] = Math.min(a[0], b[0]);
  out[1] = Math.min(a[1], b[1]);
  out[2] = Math.max(a[2], b[2]);
  out[3] = Math.max(a[3], b[3]);
  return out;
}

function intersection(out: aabb, a: Readonly<aabb>, b: Readonly<aabb>): aabb {
  out[0] = Math.max(a[0], b[0]);
  out[1] = Math.max(a[1], b[1]);
  out[2] = Math.min(a[2], b[2]);
  out[3] = Math.min(a[3], b[3]);
  return out;
}

function expand(out: aabb, a: Readonly<aabb>, amount: number): aabb {
  out[0] = a[0] - amount;
  out[1] = a[1] - amount;
  out[2] = a[2] + amount;
  out[3] = a[3] + amount;
  return out;
}

function expandByVec(out: aabb, a: Readonly<aabb>, vec: Readonly<vec2>): aabb {
  out[0] = a[0] - vec[0];
  out[1] = a[1] - vec[1];
  out[2] = a[2] + vec[0];
  out[3] = a[3] + vec[1];
  return out;
}

function translate(out: aabb, a: Readonly<aabb>, offset: Readonly<vec2>): aabb {
  out[0] = a[0] + offset[0];
  out[1] = a[1] + offset[1];
  out[2] = a[2] + offset[0];
  out[3] = a[3] + offset[1];
  return out;
}

function scale(out: aabb, a: Readonly<aabb>, factor: number): aabb {
  const centerX = (a[0] + a[2]) * 0.5;
  const centerY = (a[1] + a[3]) * 0.5;
  const halfWidth = (a[2] - a[0]) * 0.5 * factor;
  const halfHeight = (a[3] - a[1]) * 0.5 * factor;
  out[0] = centerX - halfWidth;
  out[1] = centerY - halfHeight;
  out[2] = centerX + halfWidth;
  out[3] = centerY + halfHeight;
  return out;
}

function scaleByVec(out: aabb, a: Readonly<aabb>, factor: Readonly<vec2>): aabb {
  const centerX = (a[0] + a[2]) * 0.5;
  const centerY = (a[1] + a[3]) * 0.5;
  const halfWidth = (a[2] - a[0]) * 0.5 * factor[0];
  const halfHeight = (a[3] - a[1]) * 0.5 * factor[1];
  out[0] = centerX - halfWidth;
  out[1] = centerY - halfHeight;
  out[2] = centerX + halfWidth;
  out[3] = centerY + halfHeight;
  return out;
}

function addPoint(out: aabb, a: Readonly<aabb>, point: Readonly<vec2>): aabb {
  out[0] = Math.min(a[0], point[0]);
  out[1] = Math.min(a[1], point[1]);
  out[2] = Math.max(a[2], point[0]);
  out[3] = Math.max(a[3], point[1]);
  return out;
}

function closestPoint(out: vec2, a: Readonly<aabb>, point: Readonly<vec2>): vec2 {
  out[0] = Math.max(a[0], Math.min(a[2], point[0]));
  out[1] = Math.max(a[1], Math.min(a[3], point[1]));
  return out;
}

function distanceToPoint(a: Readonly<aabb>, point: Readonly<vec2>): number {
  const dx = Math.max(0, Math.max(a[0] - point[0], point[0] - a[2]));
  const dy = Math.max(0, Math.max(a[1] - point[1], point[1] - a[3]));
  return Math.sqrt(dx * dx + dy * dy);
}

function distanceSquaredToPoint(a: Readonly<aabb>, point: Readonly<vec2>): number {
  const dx = Math.max(0, Math.max(a[0] - point[0], point[0] - a[2]));
  const dy = Math.max(0, Math.max(a[1] - point[1], point[1] - a[3]));
  return dx * dx + dy * dy;
}

function equals(a: Readonly<aabb>, b: Readonly<aabb>): boolean {
  return Math.abs(a[0] - b[0]) <= 1e-6 &&
         Math.abs(a[1] - b[1]) <= 1e-6 &&
         Math.abs(a[2] - b[2]) <= 1e-6 &&
         Math.abs(a[3] - b[3]) <= 1e-6;
}

function exactEquals(a: Readonly<aabb>, b: Readonly<aabb>): boolean {
  return a[0] === b[0] && a[1] === b[1] && a[2] === b[2] && a[3] === b[3];
}

function str(a: Readonly<aabb>): string {
  return `aabb(${a[0]}, ${a[1]}, ${a[2]}, ${a[3]})`;
}

const aabb = {
  equals,
};

