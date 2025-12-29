export interface TileRequest {
  x: number;
  y: number;
  lod: number;
}

interface TileData {
  imageData: ImageData;
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

  get(request: TileRequest): ImageData | undefined {
    const key = this.tileKey(request);
    const tileData = this.tileCache.get(key);
    return tileData?.imageData;
  }

  private async fetchTile(request: TileRequest): Promise<void> {
    const key = this.tileKey(request);
    this.pendingRequests.add(key);

    try {
      const url = `https://placehold.co/256x256?text=${request.x},${request.y},${request.lod}`;
      const response = await fetch(url);
      const blob = await response.blob();

      const image = new Image();
      image.onload = () => {
        const canvas = document.createElement('canvas');
        canvas.width = 256;
        canvas.height = 256;
        const ctx = canvas.getContext('2d');

        if (ctx) {
          ctx.drawImage(image, 0, 0);
          const imageData = ctx.getImageData(0, 0, 256, 256);

          this.tileCache.set(key, { imageData });
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
