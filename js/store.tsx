import { TILE_SIZE, TileMetadata, TileMetadataSchema } from "./schemas.js";

export interface TileRequest {
  x: number;
  y: number;
  lod: number;
  repo: string;
  committish: string;
}

interface TileData {
  metadata: TileMetadata;
  tileData: number[];
  imageBitmap: ImageBitmap;
}

export class TileStore {
  private requestedTiles: Set<string> = new Set();
  private tileCache: Map<string, TileData> = new Map();
  private pendingRequests: Set<string> = new Set();

  private tileKey(request: TileRequest): string {
    return `${request.repo}_${request.committish}_${request.x}_${request.y}_${request.lod}`;
  }

  update(requests: TileRequest[]): void {
    this.requestedTiles.clear();

    for (const request of requests) {
      const key = this.tileKey(request);
      this.requestedTiles.add(key);

      if (!this.tileCache.has(key) && !this.pendingRequests.has(key)) {
        this.fetchTile(request);
      }
    }
  }

  get(request: TileRequest): ImageBitmap | undefined {
    const key = this.tileKey(request);
    const tileData = this.tileCache.get(key);
    return tileData?.imageBitmap;
  }

  private async fetchTile(request: TileRequest): Promise<void> {
    const key = this.tileKey(request);
    this.pendingRequests.add(key);

    try {
      const url = `/api/repo/${request.repo}/${request.committish}/tile/${request.lod}/${request.x}/${request.y}/length`;
      const response = await fetch(url);
      
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const responseText = await response.text();
      const lines = responseText.split('\n');
      
      if (lines.length < 2) {
        throw new Error('Invalid response format: expected metadata line and data line');
      }

      // Parse metadata from first line
      let metadata: TileMetadata;
      try {
        metadata = TileMetadataSchema.parse(JSON.parse(lines[0]));
      } catch (error) {
        throw new Error(`Failed to parse tile metadata: ${error}`);
      }

      // Parse tile data from second line
      let tileData: number[];
      try {
        tileData = JSON.parse(lines[1]);
        if (!Array.isArray(tileData)) {
          throw new Error('Tile data is not an array');
        }
      } catch (error) {
        throw new Error(`Failed to parse tile data: ${error}`);
      }

      // Create a canvas to visualize the tile data
      const canvas = document.createElement('canvas');
      canvas.width = TILE_SIZE;
      canvas.height = TILE_SIZE;
      const ctx = canvas.getContext('2d');
      
      if (!ctx) {
        throw new Error('Failed to get canvas context');
      }

      // Create image data from tile data
      const imageData = ctx.createImageData(TILE_SIZE, TILE_SIZE);
      for (let i = 0; i < tileData.length && i < TILE_SIZE * TILE_SIZE; i++) {
        const value = Math.min(255, Math.max(0, tileData[i] || 0));
        const pixelIndex = i * 4;
        imageData.data[pixelIndex] = value;     // R
        imageData.data[pixelIndex + 1] = value; // G  
        imageData.data[pixelIndex + 2] = value; // B
        imageData.data[pixelIndex + 3] = 255;   // A
      }
      
      ctx.putImageData(imageData, 0, 0);
      
      // Convert canvas to ImageBitmap
      const imageBitmap = await createImageBitmap(canvas);
      
      this.tileCache.set(key, { 
        metadata, 
        tileData,
        imageBitmap 
      });
      
    } catch (error) {
      console.error("Failed to fetch tile:", error);
    } finally {
      this.pendingRequests.delete(key);
    }
  }
}
