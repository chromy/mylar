import { vec2, vec3 } from "gl-matrix";
import { useState, useEffect, useCallback, useRef } from "react";
import { Camera } from "./camera.js";
import { TILE_SIZE } from "./schemas.js";

function createTileImageData(): ImageData {
  let data = new ImageData(TILE_SIZE, TILE_SIZE);
  return data;
}

// On each frame:
// - Given repo, committish, bounds in world space, ops
// - 

export interface TileLayout {
  lineCount: number;
  tileCount: number;
}

interface CanvasState {
  canvas: HTMLCanvasElement;
  ctx: CanvasRenderingContext2D;
  dpr: number;
}

// Rename Viewer -> RendererHost
class Renderer {
  private repo: string;
  private committish: string;
  private layout: TileLayout;
  private camera: Camera;
  private frameId: undefined|number;
  private lastTimestampMs: number;
  private lastFrameMs: number;
  private getCanvas: () => HTMLCanvasElement|undefined;
  private canvasState: CanvasState|undefined;

  private boundFrame: (timestamp: number) => void;
  private boundHandleWheel: (e: WheelEvent) => void;
  private boundHandleResize: () => void;

  constructor(repo: string, committish: string, layout: TileLayout, getCanvas: () => HTMLCanvasElement|undefined) {
    this.repo = repo;
    this.committish = committish;
    this.layout = layout;
    this.camera = new Camera();
    this.frameId = undefined;
    this.lastTimestampMs = 0;
    this.lastFrameMs = 0;
    this.getCanvas = getCanvas;

    this.boundFrame = this.frame.bind(this);
    this.boundHandleWheel = this.handleWheel.bind(this);
    this.boundHandleResize = this.handleResize.bind(this);
  }

  private handleResize(): void {
    const canvasState = this.canvasState;
    if (canvasState === undefined) {
      return;
    }

    const {canvas, dpr} = canvasState;

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

  private tryHookCanvas(): void {
    const canvas = this.getCanvas();
    if (canvas  === undefined) {
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

    canvas.addEventListener("wheel", this.boundHandleWheel);
    window.addEventListener("resize", this.boundHandleResize);

    this.canvasState = {
      canvas,
      ctx,
      dpr,
    };
  }

  private frame(timestamp: number): void {
    this.lastFrameMs = timestamp - this.lastTimestampMs;
    this.lastTimestampMs = timestamp;


    if (this.canvasState === undefined) {
      this.tryHookCanvas();
    }

    const canvasState = this.canvasState;
    if (canvasState !== undefined) {
      const {ctx} = canvasState;
      this.renderFrame(ctx);
    }

    this.frameId = window.requestAnimationFrame(this.boundFrame);
  }

  private renderFrame(ctx: CanvasRenderingContext2D): void {
    const width = this.camera.screenWidthPx;
    const height = this.camera.screenHeightPx;

    // Clear
    ctx.clearRect(0, 0, width, height);

    // Checkerboard
    const squareSize = 4;

    // Transform the visible corners to world space using Camera
    const corners = [
      vec2.fromValues(0, 0),
      vec2.fromValues(width, 0),
      vec2.fromValues(width, height),
      vec2.fromValues(0, height),
    ];

    const worldCorners = corners.map(corner => {
      const world = vec2.create();
      this.camera.toWorld(world, corner);
      return world;
    });

    // Find bounding box in world space
    const minX = Math.min(...worldCorners.map(c => c[0]));
    const maxX = Math.max(...worldCorners.map(c => c[0]));
    const minY = Math.min(...worldCorners.map(c => c[1]));
    const maxY = Math.max(...worldCorners.map(c => c[1]));

    const startCol = Math.floor(minX / squareSize);
    const endCol = Math.ceil(maxX / squareSize);
    const startRow = Math.floor(minY / squareSize);
    const endRow = Math.ceil(maxY / squareSize);

    for (let row = startRow; row < endRow; row++) {
      for (let col = startCol; col < endCol; col++) {
        const isEven = (row + col) % 2 === 0;
        ctx.fillStyle = isEven ? "pink" : "white";

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

  start(): void {
    console.log("renderer starting")
    this.lastTimestampMs = performance.now();
    this.frameId = window.requestAnimationFrame(this.boundFrame);
  }

  stop(): void {
    console.log("renderer stopping")
    if (this.frameId !== undefined) {
      window.cancelAnimationFrame(this.frameId);
    }
    const canvasState = this.canvasState;
    if (canvasState) {
      canvasState.canvas.removeEventListener("wheel", this.boundHandleWheel);
      window.removeEventListener("resize", this.boundHandleResize);
    }
  }
}

export interface ViewerProps {
  repo: string;
  committish: string;
  layout: TileLayout;
}

export const Viewer = ({ repo, committish, layout }: ViewerProps) => {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const getCanvas = () => {
      return canvasRef.current ?? undefined;
    };

    const renderer = new Renderer(repo, committish, layout, getCanvas);
    renderer.start();

    return () => {
      renderer.stop();
    };
  }, []);

  return (
    <canvas ref={canvasRef} style={{ width: "100%", height: "100%" }}></canvas>
  );
};
