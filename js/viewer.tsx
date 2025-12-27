import { vec2, vec3 } from 'gl-matrix';
import { useState, useEffect, useCallback, useRef } from 'react';
import { Camera } from "./camera.js";

export interface IndexPanelProps {
  repo: string;
}

export const IndexPanel = ({repo}: string) => {
};

export interface ViewerProps {
  repo: string;
}

export const Viewer = ({repo}: ViewerProps) => {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const cameraRef = useRef<Camera>(null);

  const getCamera = () => {
    const camera = cameraRef.current;
    if (camera !== null) {
      return camera;
    }
    const newCamera = new Camera();
    cameraRef.current = newCamera;
    return newCamera;
  };

  const [, setRenderTrigger] = useState(0);
  const triggerRender = () => setRenderTrigger(prev => prev + 1);

  const drawCheckerboard = useCallback(() => {
    const canvas = canvasRef.current;
    if (!canvas) {
      return;
    }

    const ctx = canvas.getContext('2d');
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
    //canvas.style.width = cssWidth + 'px';
    //canvas.style.height = cssHeight + 'px';

    const camera = getCamera();
    camera.setScreenSize(vec2.fromValues(width, height));

    ctx.clearRect(0, 0, width, height);

    const squareSize = 4;

    // Transform the visible corners to world space using Camera
    const corners = [
      vec2.fromValues(0, 0),
      vec2.fromValues(width, 0),
      vec2.fromValues(width, height),
      vec2.fromValues(0, height)
    ];

    const worldCorners = corners.map(corner => {
      const world = vec2.create();
      camera.toWorld(world, corner);
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
        ctx.fillStyle = isEven ? 'pink' : 'white';

        // Convert world space square to screen space for drawing
        const worldPos = vec2.fromValues(col * squareSize, row * squareSize);
        const screenPos = vec2.create();
        camera.toScreen(screenPos, worldPos);

        const worldPosEnd = vec2.fromValues((col + 1) * squareSize, (row + 1) * squareSize);
        const screenPosEnd = vec2.create();
        camera.toScreen(screenPosEnd, worldPosEnd);

        ctx.fillRect(
          screenPos[0],
          screenPos[1],
          screenPosEnd[0] - screenPos[0],
          screenPosEnd[1] - screenPos[1]
        );
      }
    }

    // Draw black circle at world coordinates (0,0)
    const worldCenter = vec2.fromValues(0, 0);
    const screenCenter = vec2.create();
    camera.toScreen(screenCenter, worldCenter);

    const radius = 10; // Circle radius in pixels
    ctx.fillStyle = 'black';
    ctx.beginPath();
    ctx.arc(screenCenter[0], screenCenter[1], radius, 0, 2 * Math.PI);
    ctx.fill();
  }, []);

  useEffect(() => {
    drawCheckerboard();

    const handleResize = () => {
      drawCheckerboard();
    };

    const handleWheel = (e: WheelEvent) => {
      e.preventDefault();

      const camera = getCamera();

      if (e.ctrlKey || e.metaKey) {
        const zoomDelta = e.deltaY;
        camera.dolly(vec3.fromValues(0, 0, zoomDelta));
      } else {
        camera.dolly(vec3.fromValues(e.deltaX, e.deltaY, 0));
      }

      triggerRender();
    };

    const canvas = canvasRef.current;
    if (canvas) {
      canvas.addEventListener("wheel", handleWheel);
    }
    window.addEventListener("resize", handleResize);

    return () => {
      if (canvas) {
        canvas.removeEventListener("wheel", handleWheel);
      }
      window.removeEventListener("resize", handleResize);
    };
  }, [drawCheckerboard, triggerRender]);

  return (
    <canvas ref={canvasRef} style={{ width: '100%', height: '100%' }}></canvas>
  );
};


