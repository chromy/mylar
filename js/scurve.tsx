function rot(x: number): number {
  switch (x) {
    case 1:
      return 2;
    case 2:
      return 1;
    default:
      return x;
  }
}

function graycode(x: number): number {
  switch (x) {
    case 3:
      return 2;
    case 2:
      return 3;
    default:
      return x;
  }
}

/*
  Convert an index on the 2d Hilbert curve to a set of point coordinates.
*/
export function hilbertToPoint(curve: number, h: number): [number, number] {
  //  The bit widths in this function are:
  //    x, y  - curve
  //    h   - curve*2
  //    l   - 2
  //    e   - 2
  let hwidth = curve * 2;
  let e = 0;
  let d = 0;
  let x = 0;
  let y = 0;
  for (let i = 0; i < curve; i++) {
    // Extract 2 bits from h
    let w = (h >> (hwidth - i * 2 - 2)) & 3;

    let l = graycode(w);
    if (d === 0) l = rot(l);
    l = l ^ e;
    let bit = 1 << (curve - i - 1);
    if (l & 2) {
      x |= bit;
    }
    if (l & 1) {
      y |= bit;
    }

    if (w == 3) {
      e = 3 - e;
    }
    if (w === 0 || w == 3) {
      d ^= 1;
    }
  }
  return [x, y];
}

/*
  Convert a point on the Hilbert 2d curve to a set of coordinates.
*/
export function pointToHilbert(curve: number, x: number, y: number): number {
  let h = 0;
  let e = 0;
  let d = 0;
  for (let i = 0; i < curve; i++) {
    // Extract 1 bit from x and y
    let off = curve - i - 1;
    let a = (y >> off) & 1;
    let b = (x >> off) & 1;
    let l = a | (b << 1);
    l = l ^ e;
    if (d === 0) {
      l = rot(l);
    }
    let w = graycode(l);
    if (w == 3) {
      e = 3 - e;
    }
    h = (h << 2) | w;
    if (w === 0 || w == 3) {
      d ^= 1;
    }
  }
  return h;
}
