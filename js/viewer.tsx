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
import { lodToSize, lineToWorld, worldToTile, worldToLine, type WorldPosition, type TilePosition, type LinePosition } from "./utils.js";
import { type TileRequest, TileStore } from "./store.js";
import {
  type MylarAction,
  settings,
  initialMylarState,
  type MylarState,
} from "./state.js";

function boxToTileRequest(
  box: aabb,
  repo: string,
  committish: string,
  kind: string,
): TileRequest {
  const width = aabb.width(box);
  return {
    x: box[0] / width,
    y: box[1] / width,
    lod: toLod(box),
    repo,
    committish,
    kind,
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


class Renderer {
  private repo: string;
  private committish: string;
  private layout: TileLayout;
  private camera: Camera;
  private frameId: undefined | number;
  private lastTimestampMs: number;
  private lastDebugUpdateMs: number;
  private lastFrameMs: number;
  private canvasState: CanvasState | undefined;
  private callbacks: RendererHostCallbacks;
  private tileStore: TileStore;

  private boundFrame: (timestamp: number) => void;
  private boundHandleWheel: (e: WheelEvent) => void;
  private boundHandleResize: () => void;
  private boundHandleMouseMove: (e: MouseEvent) => void;
  private screenWorldAabb: aabb;
  private screenMouse: vec2;
  private worldMouse: vec2;
  private worldMousePosition: WorldPosition;
  private tilePosition: TilePosition;
  private linePosition: LinePosition;
  private visualizationBounds: aabb;

  constructor(
    repo: string,
    committish: string,
    layout: TileLayout,
    callbacks: RendererHostCallbacks,
  ) {
    this.repo = repo;
    this.committish = committish;
    this.layout = layout;
    this.camera = new Camera();
    this.frameId = undefined;
    this.lastTimestampMs = 0;
    this.lastFrameMs = 0;
    this.lastDebugUpdateMs = 0;
    this.callbacks = callbacks;
    this.screenWorldAabb = aabb.create();
    this.tileStore = new TileStore();
    this.visualizationBounds = quadtreeBoundingBox(this.layout.lineCount);

    this.boundFrame = this.frame.bind(this);
    this.boundHandleWheel = this.handleWheel.bind(this);
    this.boundHandleResize = this.handleResize.bind(this);
    this.boundHandleMouseMove = this.handleMouseMove.bind(this);
    this.screenMouse = vec2.create();
    this.worldMouse = vec2.create();
    this.worldMousePosition = { X: 0, Y: 0 };
    this.tilePosition = { Lod: 0, TileX: 0, TileY: 0, OffsetX: 0, OffsetY: 0 };
    this.linePosition = 0;
  }

  private handleResize(): void {
    const canvasState = this.canvasState;
    if (canvasState === undefined) {
      return;
    }

    const { canvas, dpr } = canvasState;

    const cssWidth = canvas.offsetWidth;
    const cssHeight = canvas.offsetHeight;
    const width = cssWidth * dpr;
    const height = cssHeight * dpr;
    canvas.width = width;
    canvas.height = height;
    this.camera.setScreenSize(vec2.fromValues(width, height));
  }

  private handleWheel(e: WheelEvent): void {
    e.preventDefault();

    if (e.ctrlKey || e.metaKey) {
      const zoomDelta = e.deltaY;
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

    // Disable all forms of image smoothing for pixel-perfect rendering
    ctx.imageSmoothingEnabled = false;
    ctx.imageSmoothingQuality = "low";

    const dpr = window.devicePixelRatio || 1;

    const cssWidth = canvas.offsetWidth;
    const cssHeight = canvas.offsetHeight;
    const width = cssWidth * dpr;
    const height = cssHeight * dpr;
    canvas.width = width;
    canvas.height = height;
    this.camera.setScreenSize(vec2.fromValues(width, height));

    const expandedBox = aabb.create();
    aabb.scale(expandedBox, this.visualizationBounds, 1.1);
    this.camera.snapToBox(expandedBox);

    canvas.addEventListener("wheel", this.boundHandleWheel);
    canvas.addEventListener("mousemove", this.boundHandleMouseMove);
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
    this.worldMousePosition = { X: this.worldMouse[0], Y: this.worldMouse[1] };
    this.tilePosition = worldToTile(this.worldMousePosition, { LastLine: this.layout.lineCount - 1 });
    this.linePosition = worldToLine(this.worldMousePosition, { LastLine: this.layout.lineCount - 1 });
    this.callbacks.setHoveredLineNumber(this.linePosition);

    const state = this.callbacks.getState();
    const pixelsPerWorldUnit =
      this.camera.screenWidthPx / aabb.width(this.screenWorldAabb);

    // If we haven't yet managed to initialize the canvas do so now:
    if (this.canvasState === undefined) {
      this.tryHookCanvas();
    }

    // Figure out what tiles ought to be ready:
    const reqs: TileRequest[] = [];
    for (const box of requiredTiles(
      this.screenWorldAabb,
      this.layout.lineCount,
      pixelsPerWorldUnit,
    )) {
      //const kind = "fileHash";
      //const kind = "offset";
      const kind = "length";
      reqs.push(boxToTileRequest(box, this.repo, this.committish, kind));
    }

    // Update tile store with required tiles
    this.tileStore.update(reqs);

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

      if (displayFps.get(state)) {
        debugItems.push(["Frame duration", this.lastFrameMs.toFixed(2) + "ms"]);
      }
      if (displayMouseDebug.get(state)) {
        debugItems.push(["World bbox", `(${x}, ${y}) (${w}, ${h})`]);
        debugItems.push(["Screen mouse", `(${screenMouseX}, ${screenMouseY})`]);
        debugItems.push(["World mouse", `(${worldMouseX}, ${worldMouseY})`]);
        debugItems.push([
          "Tile position",
          `T(${this.tilePosition.TileX}, ${this.tilePosition.TileY}) O(${this.tilePosition.OffsetX.toFixed(0)}, ${this.tilePosition.OffsetY.toFixed(0)})`
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
    request: TileRequest,
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
    requiredTileRequests: TileRequest[],
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
      const imageBitmap = this.tileStore.get(request);
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
      window.removeEventListener("resize", this.boundHandleResize);
    }
  }

  get boundingBox(): aabb {
    return this.visualizationBounds;
  }
}

export interface ViewerProps {
  repo: string;
  committish: string;
  layout: TileLayout;
  setDebug: (info: DebugInfo) => void;
  setHoveredLineNumber: (line: number) => void;
  dispatch: ActionDispatch<[action: MylarAction]>;
  state: MylarState;
}

// Rename Viewer -> RendererHost
export const Viewer = ({
  dispatch,
  state,
  repo,
  committish,
  layout,
  setDebug,
  setHoveredLineNumber,
}: ViewerProps) => {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const stateRef = useRef<MylarState>(initialMylarState);

  const [frameHistory, setFrameHistory] = useState<number[]>([]);

  useEffect(() => {
    stateRef.current = state;
  }, [state]);

  useEffect(() => {
    const getCanvas = () => {
      return canvasRef.current ?? undefined;
    };

    const getState = () => {
      return stateRef.current;
    };

    const renderer = new Renderer(repo, committish, layout, {
      getCanvas,
      setFrameHistory,
      setDebug,
      getState,
      setHoveredLineNumber,
    });
    renderer.start();

    return () => {
      renderer.stop();
    };
  }, [setFrameHistory, setDebug, setHoveredLineNumber]);

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
