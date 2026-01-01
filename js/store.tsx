import { TILE_SIZE, type TileMetadata, TileMetadataSchema } from "./schemas.js";

export interface TileRequest {
  x: number;
  y: number;
  lod: number;
  repo: string;
  kind: string;
  committish: string;
}

interface TileData {
  metadata: TileMetadata;
  tileData: number[];
  imageBitmap: ImageBitmap;
}

async function fetchTile(request: TileRequest): Promise<TileData> {
  const url = `/api/tile/${request.kind}/${request.repo}/${request.committish}/${request.lod}/${request.x}/${request.y}`;
  const response = await fetch(url);

  if (!response.ok) {
    throw new Error(`HTTP ${response.status}: ${response.statusText}`);
  }

  const responseText = await response.text();
  const lines = responseText.split("\n");

  if (lines.length < 2) {
    throw new Error(
      "Invalid response format: expected metadata line and data line",
    );
  }

  const rawMetadata = lines[0]!;
  const rawData = lines[1]!;

  let metadata: TileMetadata;
  try {
    metadata = TileMetadataSchema.parse(JSON.parse(rawMetadata));
  } catch (error) {
    throw new Error(`Failed to parse tile metadata: ${error}`);
  }

  // Parse tile data from second line
  let tileData: number[];
  try {
    tileData = JSON.parse(rawData);
    if (!Array.isArray(tileData)) {
      throw new Error("Tile data is not an array");
    }
  } catch (error) {
    throw new Error(`Failed to parse tile data: ${error}`);
  }

  const buffer = new Uint8ClampedArray(TILE_SIZE * TILE_SIZE * 4);
  for (let i = 0; i < TILE_SIZE * TILE_SIZE; i++) {
    const pixelIndex = i * 4;
    const d = tileData[i]!;

    //const value = 255 - Math.min(255, Math.max(0, tileData[i]!));
    //buffer[pixelIndex] = value; // R
    //buffer[pixelIndex + 1] = value; // G
    //buffer[pixelIndex + 2] = value; // B
    //buffer[pixelIndex + 3] = 255; // A

    const value = d === 0 ? 255 : d % 256;
    buffer[pixelIndex] = value; // R
    buffer[pixelIndex + 1] = value; // G
    buffer[pixelIndex + 2] = value; // B
    buffer[pixelIndex + 3] = 255; // A
  }

  const imageData = new ImageData(buffer, TILE_SIZE);

  const imageBitmap = await createImageBitmap(
    imageData,
    0,
    0,
    TILE_SIZE,
    TILE_SIZE,
    {
      resizeWidth: TILE_SIZE * 16,
      resizeHeight: TILE_SIZE * 16,
      premultiplyAlpha: "none",
      colorSpaceConversion: "none",
      imageOrientation: "none",
      resizeQuality: "pixelated",
    },
  );

  return {
    metadata,
    tileData,
    imageBitmap,
  };
}

export class TileStore {
  private requestedTiles: Set<string> = new Set();
  private tileCache: Map<string, TileData> = new Map();
  private pendingRequests: Set<string> = new Set();
  private requestQueue: TileRequest[] = [];
  private liveRequests: Set<string> = new Set();
  private readonly maxLiveRequests: number;

  constructor(maxLiveRequests: number = 6) {
    this.maxLiveRequests = maxLiveRequests;
  }

  private tileKey(request: TileRequest): string {
    return `${request.repo}_${request.committish}_${request.x}_${request.y}_${request.lod}`;
  }

  update(requests: TileRequest[]): void {
    this.requestedTiles.clear();

    // Remove canceled requests from queue
    this.requestQueue = this.requestQueue.filter(request => {
      const key = this.tileKey(request);
      return requests.some(r => this.tileKey(r) === key);
    });

    for (const request of requests) {
      const key = this.tileKey(request);
      this.requestedTiles.add(key);

      if (!this.tileCache.has(key) && !this.pendingRequests.has(key)) {
        if (this.liveRequests.size < this.maxLiveRequests) {
          this.requestTile(request);
        } else {
          // Add to queue if not already there
          const alreadyQueued = this.requestQueue.some(
            r => this.tileKey(r) === key,
          );
          if (!alreadyQueued) {
            this.requestQueue.push(request);
          }
        }
      }
    }
  }

  get(request: TileRequest): ImageBitmap | undefined {
    const key = this.tileKey(request);
    const tileData = this.tileCache.get(key);
    return tileData?.imageBitmap;
  }

  private async requestTile(request: TileRequest): Promise<void> {
    const key = this.tileKey(request);
    this.pendingRequests.add(key);
    this.liveRequests.add(key);

    try {
      const tile = await fetchTile(request);
      this.tileCache.set(key, tile);
    } catch (error) {
      console.error("Failed to fetch tile:", error);
    } finally {
      this.pendingRequests.delete(key);
      this.liveRequests.delete(key);
      this.processQueue();
    }
  }

  private processQueue(): void {
    while (
      this.requestQueue.length > 0 &&
      this.liveRequests.size < this.maxLiveRequests
    ) {
      const request = this.requestQueue.shift()!;
      const key = this.tileKey(request);

      // Only process if still requested and not already cached or pending
      if (
        this.requestedTiles.has(key) &&
        !this.tileCache.has(key) &&
        !this.pendingRequests.has(key)
      ) {
        this.requestTile(request);
      }
    }
  }
}
