import { mat3, vec2 } from 'gl-matrix';
import { useState, useEffect, useCallback, useRef } from 'react';
import { Camera } from "./camera.js";

export const Viewer = () => {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const cameraRef = useRef<Camera>(null);

  const getCamera = () => {
    const camera = cameraRef.current;
    if (camera !== null) {
      return camera;
    }
    const newCamera = new Camera();
    cameraRef.current = camera;
    return newCamera;
  };

  const [transform, setTransform] = useState(() => {
    const matrix = mat3.create();
    return matrix;
  });

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
    const displayWidth = canvas.offsetWidth;
    const displayHeight = canvas.offsetHeight;

    canvas.width = displayWidth * dpr;
    canvas.height = displayHeight * dpr;
    canvas.style.width = displayWidth + 'px';
    canvas.style.height = displayHeight + 'px';

    ctx.clearRect(0, 0, canvas.width, canvas.height);

    const squareSize = 50;

    // Calculate inverse transform to determine visible area
    const invTransform = mat3.create();
    mat3.invert(invTransform, transform);

    // Transform the visible corners to world space
    const corners = [
      vec2.fromValues(0, 0),
      vec2.fromValues(displayWidth, 0),
      vec2.fromValues(displayWidth, displayHeight),
      vec2.fromValues(0, displayHeight)
    ];

    corners.forEach(corner => {
      vec2.transformMat3(corner, corner, invTransform);
    });

    // Find bounding box in world space
    const minX = Math.min(...corners.map(c => c[0]));
    const maxX = Math.max(...corners.map(c => c[0]));
    const minY = Math.min(...corners.map(c => c[1]));
    const maxY = Math.max(...corners.map(c => c[1]));

    const startCol = Math.floor(minX / squareSize);
    const endCol = Math.ceil(maxX / squareSize);
    const startRow = Math.floor(minY / squareSize);
    const endRow = Math.ceil(maxY / squareSize);

    for (let row = startRow; row < endRow; row++) {
      for (let col = startCol; col < endCol; col++) {
        const isEven = (row + col) % 2 === 0;
        ctx.fillStyle = isEven ? 'pink' : 'white';
        ctx.fillRect(
          col * squareSize,
          row * squareSize,
          squareSize,
          squareSize
        );
      }
    }
  }, [transform]);

  useEffect(() => {
    drawCheckerboard();

    const handleResize = () => {
      drawCheckerboard();
    };

    const handleWheel = (e: WheelEvent) => {
      e.preventDefault();

      if (e.ctrlKey || e.metaKey) {
        // Zoom
        const canvas = canvasRef.current;
        if (!canvas) return;
        const rect = canvas.getBoundingClientRect();
        const mouseX = e.clientX - rect.left;
        const mouseY = e.clientY - rect.top;

        const zoomFactor = e.deltaY > 0 ? 0.9 : 1.1;

        setTransform(prev => {
          // Create transformation: translate to mouse, scale, translate back
          const newTransform = mat3.create();
          mat3.copy(newTransform, prev);

          // Translate to mouse position
          mat3.translate(newTransform, newTransform, [mouseX, mouseY]);

          // Scale
          mat3.scale(newTransform, newTransform, [zoomFactor, zoomFactor]);

          // Translate back
          mat3.translate(newTransform, newTransform, [-mouseX, -mouseY]);

          return newTransform;
        });
      } else {
        setTransform(prev => {
          const newTransform = mat3.create();
          mat3.copy(newTransform, prev);
          mat3.translate(newTransform, newTransform, [-e.deltaX, -e.deltaY]);
          return newTransform;
        });
      }
    };

    const canvas = canvasRef.current;
    if (canvas) {
      canvas.addEventListener('wheel', handleWheel);
    }

    window.addEventListener('resize', handleResize);

    return () => {
      if (canvas) {
        canvas.removeEventListener('wheel', handleWheel);
      }
      window.removeEventListener('resize', handleResize);
    };
  }, [drawCheckerboard]);

  return (
    <canvas ref={canvasRef} style={{ width: '100%', height: '100%' }}></canvas>
  );
};


