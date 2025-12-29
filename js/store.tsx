import { TILE_SIZE } from "./schemas.js";

export interface TileRequest {
  x: number;
  y: number;
  lod: number;
}

interface TileData {
  imageBitmap: ImageBitmap;
}

export class TileStore {
  private requestedTiles: Set<string> = new Set();
  private tileCache: Map<string, TileData> = new Map();
  private pendingRequests: Set<string> = new Set();

  private tileKey(request: TileRequest): string {
    return `${request.x}_${request.y}_${request.lod}`;
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
      const url = `https://placehold.co/${TILE_SIZE}x${TILE_SIZE}?text=${request.x},${request.y},${request.lod}`;
      const response = await fetch(url);
      const blob = await response.blob();

      const image = new Image();
      image.onload = async () => {
        try {
          const imageBitmap = await createImageBitmap(image);
          this.tileCache.set(key, { imageBitmap });
        } catch (error) {
          console.error('Failed to create ImageBitmap:', error);
        }
        this.pendingRequests.delete(key);
      };

      image.onerror = () => {
        this.pendingRequests.delete(key);
      };

      image.src = URL.createObjectURL(blob);
    } catch (error) {
      console.error('Failed to fetch tile:', error);
      this.pendingRequests.delete(key);
    }
  }
}
