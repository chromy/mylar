import { vec3 } from "gl-matrix";
import { floatToByte, convert, OKLCH, sRGB } from "@texel/color";
import { TILE_SIZE, type TileMetadata, TileMetadataSchema } from "./schemas.js";

export interface TileRequest {
  x: number;
  y: number;
  lod: number;
  repo: string;
  kind: string;
  commit: string;
}

interface TileData {
  metadata: TileMetadata;
  tileData: number[];
}

export interface CompositeTileRequest {
  x: number;
  y: number;
  lod: number;
  repo: string;
  commit: string;

  kind: string;
  composite: string;
}

async function createTileBitmap(
  composite: string,
  tileData: number[],
): Promise<ImageBitmap> {
  const oklch = [0, 0, 0];
  const rgb = [0, 0, 0];

  const buffer = new Uint8ClampedArray(TILE_SIZE * TILE_SIZE * 4);
  for (let i = 0; i < TILE_SIZE * TILE_SIZE; i++) {
    const pixelIndex = i * 4;
    const d = tileData[i]!;

    if (composite === "direct") {
      oklch[0] = 1.0 - Math.min(Math.max(d, 0), 255) / 256.0;
      oklch[1] = 0;
      oklch[2] = 0;
    } else if (composite === "hash") {
      oklch[0] = 1.0;
      oklch[1] = 1.0;
      oklch[2] = d % 360;
    } else {
    }

    convert(oklch, OKLCH, sRGB, rgb);

    buffer[pixelIndex] = floatToByte(rgb[0]!); // R
    buffer[pixelIndex + 1] = floatToByte(rgb[1]!); // G
    buffer[pixelIndex + 2] = floatToByte(rgb[2]!); // B
    buffer[pixelIndex + 3] = 255; // A
  }

  const imageData = new ImageData(buffer, TILE_SIZE);

  return await createImageBitmap(imageData, 0, 0, TILE_SIZE, TILE_SIZE, {
    resizeWidth: TILE_SIZE * 16,
    resizeHeight: TILE_SIZE * 16,
    premultiplyAlpha: "none",
    colorSpaceConversion: "none",
    imageOrientation: "none",
    resizeQuality: "pixelated",
  });
}

async function fetchTile(request: TileRequest): Promise<TileData> {
  const url = `/api/tile/${request.kind}/${request.repo}/${request.commit}/${request.lod}/${request.x}/${request.y}`;
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

  return {
    metadata,
    tileData,
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
    return `${request.repo}_${request.commit}_${request.kind}_${request.x}_${request.y}_${request.lod}`;
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

  get(request: TileRequest): TileData | undefined {
    const key = this.tileKey(request);
    return this.tileCache.get(key);
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

export class TileCompositor {
  private tileStore: TileStore;
  private requestedComposites: Set<string> = new Set();
  private compositeCache: Map<string, ImageBitmap> = new Map();
  private pendingComposites: Set<string> = new Set();

  constructor(maxLiveRequests: number = 6) {
    this.tileStore = new TileStore(maxLiveRequests);
  }

  private compositeKey(request: CompositeTileRequest): string {
    return `${request.repo}_${request.commit}_${request.kind}_${request.composite}_${request.x}_${request.y}_${request.lod}`;
  }

  private tileRequestFromComposite(request: CompositeTileRequest): TileRequest {
    return {
      x: request.x,
      y: request.y,
      lod: request.lod,
      repo: request.repo,
      commit: request.commit,
      kind: request.kind,
    };
  }

  update(requests: CompositeTileRequest[]): void {
    this.requestedComposites.clear();

    const tileRequests: TileRequest[] = [];

    for (const request of requests) {
      const key = this.compositeKey(request);
      this.requestedComposites.add(key);

      if (!this.compositeCache.has(key) && !this.pendingComposites.has(key)) {
        tileRequests.push(this.tileRequestFromComposite(request));
        this.pendingComposites.add(key);
      }
    }

    this.tileStore.update(tileRequests);

    // Process any ready tiles into composites
    this.processReadyTiles(requests);
  }

  get(request: CompositeTileRequest): ImageBitmap | undefined {
    const key = this.compositeKey(request);
    return this.compositeCache.get(key);
  }

  private async processReadyTiles(
    requests: CompositeTileRequest[],
  ): Promise<void> {
    for (const request of requests) {
      const key = this.compositeKey(request);

      if (this.pendingComposites.has(key) && !this.compositeCache.has(key)) {
        const tileRequest = this.tileRequestFromComposite(request);
        const tileData = this.tileStore.get(tileRequest);

        if (tileData) {
          try {
            const bitmap = await createTileBitmap(
              request.composite,
              tileData.tileData,
            );
            this.compositeCache.set(key, bitmap);
          } catch (error) {
            console.error("Failed to create composite bitmap:", error);
          } finally {
            this.pendingComposites.delete(key);
          }
        }
      }
    }
  }
}
