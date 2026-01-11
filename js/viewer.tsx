import { vec2, vec3 } from "gl-matrix";
import {
  type ActionDispatch,
  useState,
  useEffect,
  useCallback,
  useRef,
} from "react";
import { Camera } from "./camera.js";
import { TILE_SIZE } from "./schemas.js";
import { aabb } from "./aabb.js";
import { quadtreeBoundingBox, requiredTiles, toLod } from "./math.js";
import {
  lodToSize,
  lineToWorld,
  worldToTile,
  worldToLine,
  getGridSide,
  type WorldPosition,
  type TilePosition,
  type LinePosition,
} from "./utils.js";
import {
  type TileRequest,
  type CompositeTileRequest,
  TileStore,
  TileCompositor,
} from "./store.js";
import {
  type MylarAction,
  settings,
  initialMylarState,
  type MylarState,
  getCurrentLayer,
} from "./state.js";
import { type LayerType } from "./layers.js";

function isDrag(e: MouseEvent): boolean {
  return e.button === 1 || (e.button === 0 && e.metaKey);
}

function quadtreeToPath(quadtree: Uint8Array, maxN: number): vec2[] {
  const readMask = (n: number) => {
    return (quadtree[Math.floor(n / 2)]! >> ((n % 2) * 4)) & 0x0f;
  };

  interface QuadNode {
    x: number;
    y: number;
    size: number;
  }

  const drawNodes: QuadNode[] = [];
  const nodes: QuadNode[] = [];

  nodes.push({
    x: 0,
    y: 0,
    size: maxN,
  });

  let offset = 0;
  for (;;) {
    const node = nodes.shift();
    if (!node) {
      break;
    }

    if (node.size == 1) {
      drawNodes.push(node);
      continue;
    }

    const m = readMask(offset++);
    if (!m) {
      drawNodes.push(node);
      continue;
    }

    const hasNw = (m & 0x01) >> 0;
    const hasSw = (m & 0x02) >> 1;
    const hasSe = (m & 0x04) >> 2;
    const hasNe = (m & 0x08) >> 3;

    const nextSize = node.size / 2;

    if (hasNw) {
      nodes.push({
        x: node.x,
        y: node.y,
        size: nextSize,
      });
    }
    if (hasSw) {
      nodes.push({
        x: node.x,
        y: node.y + nextSize,
        size: nextSize,
      });
    }
    if (hasSe) {
      nodes.push({
        x: node.x + nextSize,
        y: node.y + nextSize,
        size: nextSize,
      });
    }
    if (hasNe) {
      nodes.push({
        x: node.x + nextSize,
        y: node.y,
        size: nextSize,
      });
    }
  }

  type Edge = [vec2, vec2];

  const idToNode = new Map<string, vec2>();
  const idToEdge = new Map<string, Edge>();

  const nodeToId = (n: vec2) => `${n[0]}_${n[1]}`;
  const edgeToId = (a: vec2, b: vec2) => `${a[0]}_${a[1]}_${b[0]}_${b[1]}`;

  const internNode = (x: number, y: number) => {
    const id = `${x}_${y}`;
    const node = idToNode.get(id);
    if (node !== undefined) {
      return node;
    }

    const newNode = vec2.fromValues(x, y);
    idToNode.set(id, newNode);

    return newNode;
  };

  const internEdge = (a: vec2, b: vec2) => {
    const id = `${nodeToId(a)}_${nodeToId(b)}`;
    const edge = idToEdge.get(id);
    if (edge !== undefined) {
      return edge;
    }

    const newEdge: Edge = [a, b];
    idToEdge.set(id, newEdge);

    return newEdge;
  };

  const reverseEdge = (e: [vec2, vec2]) => internEdge(e[1], e[0]);

  interface Interval {
    a: number;
    b: number;
    s: number;
  }

  const rows = new Map<number, Interval[]>();
  const getRow = (y: number) => {
    const row = rows.get(y);
    if (row !== undefined) {
      return row;
    }
    const newRow: Interval[] = [];
    rows.set(y, newRow);
    return newRow;
  };

  const columns = new Map<number, Interval[]>();
  const getColumn = (x: number) => {
    const column = columns.get(x);
    if (column !== undefined) {
      return column;
    }
    const newColumn: Interval[] = [];
    columns.set(x, newColumn);
    return newColumn;
  };

  for (const { x, y, size } of drawNodes) {
    getColumn(x).push({
      a: y,
      b: y + size,
      s: 1,
    });
    getColumn(x + size).push({
      a: y,
      b: y + size,
      s: -1,
    });

    getRow(y).push({
      a: x,
      b: x + size,
      s: -1,
    });
    getRow(y + size).push({
      a: x,
      b: x + size,
      s: 1,
    });
  }

  const boundary: Edge[] = [];
  const processAxis = (
    n: number,
    intervals: Interval[],
    isVertical: boolean,
  ) => {
    const points = [...new Set(intervals.flatMap(({ a, b }) => [a, b]))];
    points.sort((a, b) => a - b);

    for (let i = 0; i < points.length - 1; i++) {
      const k = points[i]!;
      const j = points[i + 1]!;
      const mid = (k + j) / 2;

      let winding = 0;
      for (const interval of intervals) {
        if (interval.a <= mid && interval.b > mid) {
          winding += interval.s;
        }
      }

      const p = isVertical ? internNode(n, k) : internNode(k, n);
      const q = isVertical ? internNode(n, j) : internNode(j, n);

      if (winding === 0) {
        // do nothing.
      } else if (winding > 0) {
        boundary.push(internEdge(p, q));
      } else {
        boundary.push(internEdge(q, p));
      }
    }
  };

  rows.entries().forEach(([y, intervals]) => processAxis(y, intervals, false));
  columns
    .entries()
    .forEach(([x, intervals]) => processAxis(x, intervals, true));

  const adj = new Map<vec2, vec2>();
  for (const e of boundary) {
    if (adj.has(e[0])) {
      debugger;
    }
    adj.set(e[0], e[1]);
  }

  const points: vec2[] = [];
  const start = [...boundary][0]![0];
  let node = start;
  for (;;) {
    points.push(node);
    node = adj.get(node)!;
    if (node === start) {
      break;
    }
  }

  return points;

  //return [...boundary].flatMap(x => x);

  //const nw = internNode(node.x, node.y);
  //const sw = internNode(node.x, node.y+node.size);
  //const se = internNode(node.x+node.size, node.y+node.size);
  //const ne = internNode(node.x+node.size, node.y);

  //const left = internEdge(nw, sw);
  //const bottom = internEdge(sw, se);
  //const right = internEdge(se, ne);
  //const top = internEdge(ne, nw);
  //const edges = [left, bottom, right, top];

  //for (const edge of edges) {
  //  const r = reverseEdge(edge);
  //  if (boundary.has(r)) {
  //    boundary.delete(r);
  //  } else {
  //    boundary.add(edge);
  //  }
  //}

  //const points = [];

  //const start = [...boundary][0]![0];
  //let node = start;
  //for (;;) {
  //  debugger;
  //  points.push(node);
  //  node = adj.get(node)!;
  //  if (node === start) {
  //    break;
  //  }
  //}

  //return points;
}

function boxToCompositeTileRequest(
  box: aabb,
  repo: string,
  commit: string,

  kind: string,
  composite: string,
  aggregation: string,
): CompositeTileRequest {
  const width = aabb.width(box);
  return {
    x: box[0] / width,
    y: box[1] / width,
    lod: toLod(box),
    repo,
    commit,
    kind,
    composite,
    aggregation,
  };
}

export interface TileLayout {
  lineCount: number;
  tileCount: number;
}

export type DebugKeyValue = [string, string];
export type DebugInfo = DebugKeyValue[];

interface CanvasState {
  canvas: HTMLCanvasElement;
  ctx: CanvasRenderingContext2D;
  dpr: number;
}

interface RendererHostCallbacks {
  getCanvas: () => HTMLCanvasElement | undefined;
  setDebug: (info: DebugInfo) => void;
  setFrameHistory: (history: number[]) => void;
  setHoveredLineNumber: (line: number) => void;
  getState(): MylarState;
  getHoveredOutline(): Uint8Array | undefined;
}

const displayOrigin = settings.addBoolean({
  id: "setting.displayOrigin",
  name: "origin",
});

const displayFps = settings.addBoolean({
  id: "setting.displayFps",
  name: "FPS",
});

const displayTileBorders = settings.addBoolean({
  id: "setting.displayTileBorders",
  name: "tile borders",
});

const displayBoundingBox = settings.addBoolean({
  id: "setting.displayBoundingBox",
  name: "bounding box",
  defaultValue: true,
});

const displayMouseDebug = settings.addBoolean({
  id: "setting.displayMouseDebug",
  name: "mouse debug",
  defaultValue: true,
});

const debugCurve = settings.addBoolean({
  id: "setting.debugCurve",
  name: "debug curve",
  defaultValue: false,
});

const displayFileOutline = settings.addBoolean({
  id: "setting.displayFileOutline",
  name: "file outline",
  defaultValue: true,
});

class Renderer {
  private repo: string;
  private commit: string;
  private layout: TileLayout;
  private camera: Camera;
  private frameId: undefined | number;
  private lastTimestampMs: number;
  private lastDebugUpdateMs: number;
  private lastFrameMs: number;
  private canvasState: CanvasState | undefined;
  private callbacks: RendererHostCallbacks;
  private tileStore: TileStore;
  private tileCompositor: TileCompositor;

  private boundFrame: (timestamp: number) => void;
  private boundHandleWheel: (e: WheelEvent) => void;
  private boundHandleResize: () => void;
  private boundHandleMouseMove: (e: MouseEvent) => void;
  private boundHandleMouseDown: (e: MouseEvent) => void;
  private boundHandleMouseUp: (e: MouseEvent) => void;
  private screenWorldAabb: aabb;
  private screenMouse: vec2;
  private worldMouse: vec2;
  private worldMousePosition: WorldPosition;
  private tilePosition: TilePosition;
  private linePosition: LinePosition;
  private visualizationBounds: aabb;
  private isDragging: boolean;
  private dragStartScreen: vec2;
  private cachedQuadtreePath: vec2[] | undefined;
  private lastHoveredOutline: Uint8Array | undefined;

  constructor(
    repo: string,
    commit: string,
    layout: TileLayout,
    callbacks: RendererHostCallbacks,
  ) {
    this.repo = repo;
    this.commit = commit;
    this.layout = layout;
    this.camera = new Camera();
    this.frameId = undefined;
    this.lastTimestampMs = 0;
    this.lastFrameMs = 0;
    this.lastDebugUpdateMs = 0;
    this.callbacks = callbacks;
    this.screenWorldAabb = aabb.create();
    this.tileStore = new TileStore();
    this.tileCompositor = new TileCompositor(this.tileStore);
    this.visualizationBounds = quadtreeBoundingBox(this.layout);

    this.boundFrame = this.frame.bind(this);
    this.boundHandleWheel = this.handleWheel.bind(this);
    this.boundHandleResize = this.handleResize.bind(this);
    this.boundHandleMouseMove = this.handleMouseMove.bind(this);
    this.boundHandleMouseDown = this.handleMouseDown.bind(this);
    this.boundHandleMouseUp = this.handleMouseUp.bind(this);
    this.screenMouse = vec2.create();
    this.worldMouse = vec2.create();
    this.worldMousePosition = { x: 0, y: 0 };
    this.tilePosition = { lod: 0, tileX: 0, tileY: 0, offsetX: 0, offsetY: 0 };
    this.linePosition = -1;
    this.isDragging = false;
    this.dragStartScreen = vec2.create();
    this.cachedQuadtreePath = undefined;
    this.lastHoveredOutline = undefined;
  }

  private handleResize(): void {
    const canvasState = this.canvasState;
    if (canvasState === undefined) {
      return;
    }

    const { ctx, canvas, dpr } = canvasState;

    const cssWidth = canvas.offsetWidth;
    const cssHeight = canvas.offsetHeight;
    const width = cssWidth * dpr;
    const height = cssHeight * dpr;
    canvas.width = width;
    canvas.height = height;
    ctx.imageSmoothingEnabled = false;
    ctx.imageSmoothingQuality = "low";
    this.camera.setScreenSize(vec2.fromValues(width, height));
  }

  private handleWheel(e: WheelEvent): void {
    e.preventDefault();

    if (e.ctrlKey || e.metaKey) {
      // Amplify zoom sensitivity for pinch gestures while keeping regular Cmd+scroll unchanged
      // Pinch zoom typically has much smaller deltaY values, so we amplify them
      const zoomDelta = e.deltaY * (Math.abs(e.deltaY) < 10 ? 5 : 1);
      this.camera.dolly(vec3.fromValues(0, 0, zoomDelta));
    } else {
      this.camera.dolly(vec3.fromValues(e.deltaX, e.deltaY, 0));
    }
  }

  private handleMouseMove(e: MouseEvent): void {
    const canvasState = this.canvasState;
    if (canvasState === undefined) {
      return;
    }

    const rect = canvasState.canvas.getBoundingClientRect();
    const x = (e.clientX - rect.left) * canvasState.dpr;
    const y = (e.clientY - rect.top) * canvasState.dpr;

    vec2.set(this.screenMouse, x, y);
    this.camera.toWorld(this.worldMouse, this.screenMouse);

    if (this.isDragging) {
      const deltaX = x - this.dragStartScreen[0];
      const deltaY = y - this.dragStartScreen[1];

      this.camera.dolly(vec3.fromValues(-deltaX, -deltaY, 0));

      vec2.set(this.dragStartScreen, x, y);
    }
  }

  private handleMouseDown(e: MouseEvent): void {
    const canvasState = this.canvasState;
    if (canvasState === undefined) {
      return;
    }

    if (isDrag(e)) {
      const rect = canvasState.canvas.getBoundingClientRect();
      const x = (e.clientX - rect.left) * canvasState.dpr;
      const y = (e.clientY - rect.top) * canvasState.dpr;

      this.isDragging = true;
      vec2.set(this.dragStartScreen, x, y);
      e.preventDefault();
    }
  }

  private handleMouseUp(e: MouseEvent): void {
    this.isDragging = false;
    e.preventDefault();
  }

  private tryHookCanvas(): void {
    const canvas = this.callbacks.getCanvas();
    if (canvas === undefined) {
      return;
    }

    const ctx = canvas.getContext("2d");
    if (!ctx) {
      return;
    }

    const dpr = window.devicePixelRatio || 1;

    const cssWidth = canvas.offsetWidth;
    const cssHeight = canvas.offsetHeight;
    const width = cssWidth * dpr;
    const height = cssHeight * dpr;
    canvas.width = width;
    canvas.height = height;
    this.camera.setScreenSize(vec2.fromValues(width, height));

    // Disable all forms of image smoothing for pixel-perfect rendering
    ctx.imageSmoothingEnabled = false;
    ctx.imageSmoothingQuality = "low";

    const expandedBox = aabb.create();
    aabb.scale(expandedBox, this.visualizationBounds, 1.1);
    this.camera.snapToBox(expandedBox);

    canvas.addEventListener("wheel", this.boundHandleWheel);
    canvas.addEventListener("mousemove", this.boundHandleMouseMove);
    canvas.addEventListener("mousedown", this.boundHandleMouseDown);
    canvas.addEventListener("mouseup", this.boundHandleMouseUp);
    window.addEventListener("resize", this.boundHandleResize);

    this.canvasState = {
      canvas,
      ctx,
      dpr,
    };
  }

  private frame(timestamp: number): void {
    // Frame bookkeeping:
    this.lastFrameMs = timestamp - this.lastTimestampMs;
    this.lastTimestampMs = timestamp;
    this.camera.intoWorldBoundingBox(this.screenWorldAabb);
    this.worldMousePosition = {
      x: Math.floor(this.worldMouse[0]),
      y: Math.floor(this.worldMouse[1]),
    };
    this.tilePosition = worldToTile(this.worldMousePosition, this.layout);
    this.linePosition = worldToLine(this.worldMousePosition, this.layout);
    if (!aabb.containsPoint(this.visualizationBounds, this.worldMouse)) {
      this.linePosition = -1;
    }

    this.callbacks.setHoveredLineNumber(this.linePosition);

    // Update cached quadtree path if hoveredOutline changed
    const hoveredOutline = this.callbacks.getHoveredOutline();
    if (hoveredOutline !== this.lastHoveredOutline) {
      this.lastHoveredOutline = hoveredOutline;
      if (hoveredOutline) {
        const maxN = getGridSide(this.layout);
        this.cachedQuadtreePath = quadtreeToPath(hoveredOutline, maxN);
      } else {
        this.cachedQuadtreePath = undefined;
      }
    }

    const state = this.callbacks.getState();
    const currentLayer = getCurrentLayer(state);
    const pixelsPerWorldUnit =
      this.camera.screenWidthPx / aabb.width(this.screenWorldAabb);

    // If we haven't yet managed to initialize the canvas do so now:
    if (this.canvasState === undefined) {
      this.tryHookCanvas();
    }

    // Figure out what tiles ought to be ready:
    const reqs: CompositeTileRequest[] = [];
    for (const box of requiredTiles(
      this.screenWorldAabb,
      this.layout,
      pixelsPerWorldUnit,
    )) {
      const { kind, composite, aggregation } = currentLayer;
      reqs.push(
        boxToCompositeTileRequest(
          box,
          this.repo,
          this.commit,
          kind,
          composite,
          aggregation,
        ),
      );
    }

    // Update tile compositor with required composite tiles
    this.tileCompositor.update(reqs);

    if (timestamp - this.lastDebugUpdateMs > 100) {
      this.lastDebugUpdateMs = timestamp;
      const x = Math.round(this.screenWorldAabb[0]).toString().padStart(4);
      const y = Math.round(this.screenWorldAabb[1]).toString().padStart(4);
      const w = Math.round(this.screenWorldAabb[2]).toString().padStart(4);
      const h = Math.round(this.screenWorldAabb[3]).toString().padStart(4);
      const screenMouseX = Math.round(this.screenMouse[0])
        .toString()
        .padStart(4);
      const screenMouseY = Math.round(this.screenMouse[1])
        .toString()
        .padStart(4);
      const worldMouseX = Math.round(this.worldMouse[0]).toString().padStart(4);
      const worldMouseY = Math.round(this.worldMouse[1]).toString().padStart(4);

      const debugItems: DebugInfo = [];

      debugItems.push(["# Tile requests", `${reqs.length}`]);
      debugItems.push([
        "# Outstanding fetches",
        `${this.tileStore.getLiveRequestCount()}`,
      ]);
      debugItems.push([
        "# Outstanding composites",
        `${this.tileCompositor.getProcessingJobCount()}`,
      ]);

      if (displayFps.get(state)) {
        debugItems.push(["Frame duration", this.lastFrameMs.toFixed(2) + "ms"]);
      }
      if (displayMouseDebug.get(state)) {
        debugItems.push(["World bbox", `(${x}, ${y}) (${w}, ${h})`]);
        debugItems.push(["Screen mouse", `(${screenMouseX}, ${screenMouseY})`]);
        debugItems.push(["World mouse", `(${worldMouseX}, ${worldMouseY})`]);
        debugItems.push([
          "Tile position",
          `T(${this.tilePosition.tileX}, ${this.tilePosition.tileY}) O(${this.tilePosition.offsetX.toFixed(0)}, ${this.tilePosition.offsetY.toFixed(0)})`,
        ]);
        debugItems.push(["Line position", `${this.linePosition}`]);
        debugItems.push([
          "Pixel per world unit",
          `${pixelsPerWorldUnit.toFixed(2)}`,
        ]);
      }
      this.callbacks.setDebug(debugItems);
    }

    // Render tiles which are ready:
    const canvasState = this.canvasState;
    if (canvasState !== undefined) {
      const { ctx } = canvasState;
      this.renderFrame(ctx, reqs);
    }
    // Do as much computation as fits in budget:
    // TODO

    // Schedule next frame:
    this.frameId = window.requestAnimationFrame(this.boundFrame);
  }

  private renderAABB(ctx: CanvasRenderingContext2D, aabb: aabb): void {
    const topLeft = vec2.fromValues(aabb[0], aabb[1]);
    const bottomRight = vec2.fromValues(aabb[2], aabb[3]);

    const screenTopLeft = vec2.create();
    const screenBottomRight = vec2.create();

    this.camera.toScreen(screenTopLeft, topLeft);
    this.camera.toScreen(screenBottomRight, bottomRight);

    const width = screenBottomRight[0] - screenTopLeft[0];
    const height = screenBottomRight[1] - screenTopLeft[1];

    ctx.strokeRect(screenTopLeft[0], screenTopLeft[1], width, height);
  }

  private renderTile(
    ctx: CanvasRenderingContext2D,
    request: CompositeTileRequest,
    imageBitmap: ImageBitmap,
  ): void {
    const worldSize = lodToSize(request.lod);
    const worldX = request.x * worldSize;
    const worldY = request.y * worldSize;

    const worldA = vec2.fromValues(worldX, worldY);
    const worldB = vec2.fromValues(worldX + worldSize, worldY + worldSize);

    const screenA = vec2.create();
    const screenB = vec2.create();

    this.camera.toScreen(screenA, worldA);
    this.camera.toScreen(screenB, worldB);

    // Round to pixel boundaries to prevent sub-pixel rendering blur
    const screenMinX = Math.round(Math.min(screenA[0], screenB[0]));
    const screenMinY = Math.round(Math.min(screenA[1], screenB[1]));
    const screenMaxX = Math.round(Math.max(screenA[0], screenB[0]));
    const screenMaxY = Math.round(Math.max(screenA[1], screenB[1]));

    ctx.drawImage(
      imageBitmap,
      screenMinX,
      screenMinY,
      screenMaxX - screenMinX,
      screenMaxY - screenMinY,
    );
  }

  private renderFrame(
    ctx: CanvasRenderingContext2D,
    requiredTileRequests: CompositeTileRequest[],
  ): void {
    const state = this.callbacks.getState();
    const width = this.camera.screenWidthPx;
    const height = this.camera.screenHeightPx;

    // Clear
    ctx.clearRect(0, 0, width, height);

    // Checkerboard
    const squareSize = 4;

    // Use pre-computed world bounding box
    const minX = this.screenWorldAabb[0];
    const minY = this.screenWorldAabb[1];
    const maxX = this.screenWorldAabb[2];
    const maxY = this.screenWorldAabb[3];

    const startCol = Math.floor(minX / squareSize);
    const endCol = Math.ceil(maxX / squareSize);
    const startRow = Math.floor(minY / squareSize);
    const endRow = Math.ceil(maxY / squareSize);
    const squareCount = (endCol - startCol) * (endRow - startRow);
    const zoomFadeStart = 20;
    const zoomFadeEnd = 400;
    const currentZ = this.camera.eye[2];

    let opacity = 1;
    if (currentZ > zoomFadeStart) {
      opacity = Math.max(
        0,
        1 - (currentZ - zoomFadeStart) / (zoomFadeEnd - zoomFadeStart),
      );
    }

    if (opacity >= 0.01) {
      ctx.fillStyle = `rgba(245, 245, 245, ${opacity})`;

      for (let row = startRow; row < endRow; row++) {
        for (let col = startCol; col < endCol; col++) {
          const isEven = (row + col) % 2 === 0;
          if (isEven) {
            continue;
          }

          // Convert world space square to screen space for drawing
          const worldPos = vec2.fromValues(col * squareSize, row * squareSize);
          const screenPos = vec2.create();
          this.camera.toScreen(screenPos, worldPos);

          const worldPosEnd = vec2.fromValues(
            (col + 1) * squareSize,
            (row + 1) * squareSize,
          );
          const screenPosEnd = vec2.create();
          this.camera.toScreen(screenPosEnd, worldPosEnd);

          ctx.fillRect(
            screenPos[0],
            screenPos[1],
            screenPosEnd[0] - screenPos[0],
            screenPosEnd[1] - screenPos[1],
          );
        }
      }
    }

    for (const request of requiredTileRequests) {
      const imageBitmap = this.tileCompositor.get(request);
      if (imageBitmap) {
        this.renderTile(ctx, request, imageBitmap);
      }
    }

    if (displayOrigin.get(state)) {
      // Draw black circle at world coordinates (0,0)
      const worldCenter = vec2.fromValues(0, 0);
      const screenCenter = vec2.create();
      this.camera.toScreen(screenCenter, worldCenter);
      const radius = 10; // Circle radius in pixels
      ctx.fillStyle = "black";
      ctx.beginPath();
      ctx.arc(screenCenter[0], screenCenter[1], radius, 0, 2 * Math.PI);
      ctx.fill();
    }

    if (displayTileBorders.get(state)) {
      ctx.lineWidth = 5;
      let count = 0;
      for (const r of requiredTileRequests) {
        const hue = ((count * 360) / requiredTileRequests.length) % 360;
        ctx.strokeStyle = `hsl(${hue}, 70%, 50%)`;
        const size = lodToSize(r.lod);
        const box = aabb.fromValues(
          r.x * size,
          r.y * size,
          (r.x + 1) * size,
          (r.y + 1) * size,
        );
        this.renderAABB(ctx, box);
        count++;
      }
    }

    if (displayBoundingBox.get(state)) {
      // Draw visualization bounding box as 1px black line
      ctx.strokeStyle = "black";
      ctx.lineWidth = 1;
      this.renderAABB(ctx, this.visualizationBounds);
    }

    if (
      debugCurve.get(state) &&
      aabb.containsPoint(this.visualizationBounds, this.worldMouse)
    ) {
      this.renderDebugCurve(ctx, state);
    }

    if (this.cachedQuadtreePath && displayFileOutline.get(state)) {
      this.renderQuadtree(ctx);
    }
  }

  private renderDebugCurve(
    ctx: CanvasRenderingContext2D,
    state: MylarState,
  ): void {
    const numLines = 45;
    const startLine = this.linePosition;

    ctx.strokeStyle = "hotpink";
    ctx.lineWidth = 2;
    ctx.fillStyle = "hotpink";

    for (let i = 0; i < numLines - 1; i++) {
      const currentLine = startLine + i;
      const nextLine = startLine + i + 1;

      // Skip if we exceed the total line count
      if (nextLine >= this.layout.lineCount) {
        break;
      }

      const currentWorld = lineToWorld(currentLine, {
        lineCount: this.layout.lineCount,
      });
      const nextWorld = lineToWorld(nextLine, {
        lineCount: this.layout.lineCount,
      });

      // Convert world positions to screen coordinates
      const currentWorldVec = vec2.fromValues(
        currentWorld.x + 0.5,
        currentWorld.y + 0.5,
      );
      const nextWorldVec = vec2.fromValues(
        nextWorld.x + 0.5,
        nextWorld.y + 0.5,
      );

      const currentScreen = vec2.create();
      const nextScreen = vec2.create();

      this.camera.toScreen(currentScreen, currentWorldVec);
      this.camera.toScreen(nextScreen, nextWorldVec);

      // Draw line
      ctx.beginPath();
      ctx.moveTo(currentScreen[0], currentScreen[1]);
      ctx.lineTo(nextScreen[0], nextScreen[1]);
      ctx.stroke();

      // Draw arrow head
      const angle = Math.atan2(
        nextScreen[1] - currentScreen[1],
        nextScreen[0] - currentScreen[0],
      );
      const arrowLength = 10;
      const arrowAngle = Math.PI / 6; // 30 degrees

      ctx.beginPath();
      ctx.moveTo(nextScreen[0], nextScreen[1]);
      ctx.lineTo(
        nextScreen[0] - arrowLength * Math.cos(angle - arrowAngle),
        nextScreen[1] - arrowLength * Math.sin(angle - arrowAngle),
      );
      ctx.moveTo(nextScreen[0], nextScreen[1]);
      ctx.lineTo(
        nextScreen[0] - arrowLength * Math.cos(angle + arrowAngle),
        nextScreen[1] - arrowLength * Math.sin(angle + arrowAngle),
      );
      ctx.stroke();
    }
  }

  private renderQuadtree(ctx: CanvasRenderingContext2D): void {
    const points = this.cachedQuadtreePath;
    if (!points || points.length === 0) {
      return;
    }

    ctx.strokeStyle = "rgba(0, 0, 0, 1)";
    ctx.lineWidth = 3;

    const start = points[0]!;

    const s = vec2.fromValues(0, 0);
    this.camera.toScreen(s, start);
    ctx.beginPath();
    ctx.moveTo(s[0], s[1]);
    for (let i = 1; i < points.length; ++i) {
      const point = points[i]!;
      this.camera.toScreen(s, point);
      ctx.lineTo(s[0], s[1]);
    }
    this.camera.toScreen(s, start);
    ctx.lineTo(s[0], s[1]);
    ctx.stroke();
  }

  start(): void {
    console.log("renderer starting");
    this.lastTimestampMs = performance.now();
    this.frameId = window.requestAnimationFrame(this.boundFrame);
  }

  stop(): void {
    console.log("renderer stopping");
    if (this.frameId !== undefined) {
      window.cancelAnimationFrame(this.frameId);
    }
    const canvasState = this.canvasState;
    if (canvasState) {
      canvasState.canvas.removeEventListener("wheel", this.boundHandleWheel);
      canvasState.canvas.removeEventListener(
        "mousemove",
        this.boundHandleMouseMove,
      );
      canvasState.canvas.removeEventListener(
        "mousedown",
        this.boundHandleMouseDown,
      );
      canvasState.canvas.removeEventListener(
        "mouseup",
        this.boundHandleMouseUp,
      );
      window.removeEventListener("resize", this.boundHandleResize);
    }
  }

  get boundingBox(): aabb {
    return this.visualizationBounds;
  }
}

export interface ViewerProps {
  repo: string;
  commit: string;
  tree: string;
  layout: TileLayout;
  setDebug: (info: DebugInfo) => void;
  setHoveredLineNumber: (line: number) => void;
  hoveredOutline: Uint8Array | undefined;
  dispatch: ActionDispatch<[action: MylarAction]>;
  state: MylarState;
}

// Rename Viewer -> RendererHost
export const Viewer = ({
  dispatch,
  state,
  repo,
  commit,
  tree,
  layout,
  setDebug,
  setHoveredLineNumber,
  hoveredOutline,
}: ViewerProps) => {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const stateRef = useRef<MylarState>(initialMylarState);
  const hoveredOutlineRef = useRef<Uint8Array | undefined>(undefined);

  const [frameHistory, setFrameHistory] = useState<number[]>([]);

  useEffect(() => {
    stateRef.current = state;
  }, [state]);

  useEffect(() => {
    hoveredOutlineRef.current = hoveredOutline;
  }, [hoveredOutline]);

  useEffect(() => {
    const getCanvas = () => {
      return canvasRef.current ?? undefined;
    };

    const getState = () => {
      return stateRef.current;
    };

    const getHoveredOutline = () => {
      return hoveredOutlineRef.current;
    };

    const renderer = new Renderer(repo, commit, layout, {
      getCanvas,
      setFrameHistory,
      setDebug,
      getState,
      setHoveredLineNumber,
      getHoveredOutline,
    });
    (window as any).renderer = renderer;
    renderer.start();

    return () => {
      renderer.stop();
    };
  }, [setFrameHistory, setDebug, setHoveredLineNumber, layout, commit]);

  return (
    <canvas
      ref={canvasRef}
      style={{
        width: "100%",
        height: "100%",
      }}
    ></canvas>
  );
};
