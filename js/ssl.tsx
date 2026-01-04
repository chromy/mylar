import { convert, OKLCH, sRGB, floatToByte } from "@texel/color";

export class Stack {
  private readonly capacity: number;
  private size: number;
  private items: number[];

  constructor(capacity: number) {
    this.capacity = capacity;
    this.size = 0;
    this.items = new Array(capacity).fill(0);
  }

  push(n: number): void {
    this.items[this.size++] = n;
  }

  pop(): number {
    return this.items[--this.size]!;
  }

  peek(): number {
    return this.items[this.size - 1]!;
  }

  clear(): void {
    this.size = 0;
  }
}

type Op = (stack: Stack) => void;

function binOp(f: (a: number, b: number) => number): Op {
  return stack => {
    const b = stack.pop();
    const a = stack.pop();
    const c = f(a, b);
    stack.push(c);
  };
}

function unOp(f: (x: number) => number): Op {
  return stack => {
    const x = stack.pop();
    const y = f(x);
    stack.push(y);
  };
}

function zeroOp(f: () => number): Op {
  return stack => {
    stack.push(f());
  };
}

export const constant = (n: number) => zeroOp(() => n);
export const mapTopThree = (f: (n: number) => number) => {
  return (stack: Stack) => {
    const a = stack.pop();
    const b = stack.pop();
    const c = stack.pop();
    stack.push(f(c));
    stack.push(f(b));
    stack.push(f(a));
  };
};

export const max = binOp((a, b) => Math.max(a, b));
export const min = binOp((a, b) => Math.min(a, b));
export const add = binOp((a, b) => a + b);
export const sub = binOp((a, b) => a - b);
export const mul = binOp((a, b) => a * b);
export const mod = binOp((a, b) => a % b);
export const div = binOp((a, b) => a / b);
export const toByte = unOp(floatToByte);
export const int32ToUnit = unOp(n => (n + 2147483648) / 4294967295);
export const hash = unOp(n => {
  n ^= n >>> 16;
  n = Math.imul(n, 0x85ebca6b);
  n ^= n >>> 13;
  n = Math.imul(n, 0xc2b2ae35);
  n >>> 16;
  return n;
});

export const dup = (stack: Stack) => stack.push(stack.peek());
export const swap = (stack: Stack) => {
  const a = stack.pop();
  const b = stack.pop();
  stack.push(a);
  stack.push(b);
};

export const oklchToSrgb = (stack: Stack) => {
  const h = stack.pop();
  const c = stack.pop();
  const l = stack.pop();
  const input = [l, c, h];
  const output = [0, 0, 0];
  convert(input, OKLCH, sRGB, output);
  const [r, g, b] = output;
  stack.push(r!);
  stack.push(g!);
  stack.push(b!);
};

export const rainbow = (stack: Stack) => {
  const t = stack.pop();
  const ts = Math.abs(t - 0.5);
  stack.push(1.5 - 1.5 * ts);
  stack.push(0.8 - 0.9 * ts);
  stack.push(360 * t - 100);
};

export const ops: { [key: string]: Op } = {
  swap,
  add,
  sub,
  mul,
  min,
  div,
  mod,
  dup,
  hash,
  toByte,
  toByteX3: mapTopThree(floatToByte),
  int32ToUnit,
  max,
  oklchToSrgb,
  rainbow,
};

function compilePartToOp(part: string): Op {
  const op = ops[part];
  if (op !== undefined) {
    return op;
  }
  const n = parseInt(part);
  if (isNaN(n)) {
    throw new Error(`Unknown part ${part}`);
  }
  return constant(n);
}

export function compile(program: string): Op[] {
  const parts = program.split("|");
  return parts.map(compilePartToOp);
}
