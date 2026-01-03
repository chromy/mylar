import { vec3 } from "gl-matrix";
import { floatToByte, convert, OKLCH, sRGB } from "@texel/color";
import { TILE_SIZE, type TileMetadata, TileMetadataSchema } from "./schemas.js";

const DEFAULT_MAX_LIVE_REQUESTS = 6;

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
  data: number[];
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

    if (d === 0) {
      buffer[pixelIndex+0] = 255;
      buffer[pixelIndex+1] = 255;
      buffer[pixelIndex+2] = 255;
      buffer[pixelIndex+3] = 255;
      continue;
    }

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
    resizeWidth: TILE_SIZE * 4,
    resizeHeight: TILE_SIZE * 4,
    premultiplyAlpha: "none",
    colorSpaceConversion: "none",
    imageOrientation: "none",
    resizeQuality: "pixelated",
  });
}

async function fetchTile(url: string, signal: AbortSignal): Promise<TileData> {
  const response = await fetch(url, {signal});

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
  let data: number[];
  try {
    data = JSON.parse(rawData);
    if (!Array.isArray(data)) {
      throw new Error("Tile data is not an array");
    }
  } catch (error) {
    throw new Error(`Failed to parse tile data: ${error}`);
  }

  return {
    metadata,
    data,
  };
}



export class TileStore {
  private readonly maxLiveRequests: number;

  private tiles: Map<string, TileData> = new Map();
  private cache: Map<string, WeakRef<TileData>> = new Map();

  private queue: string[] = [];
  private requested: Set<string> = new Set();
  private live: Set<string> = new Set();
  private aborts: Map<string, () => void> = new Map();
  private preventRequestsTill: number;

  constructor(maxLiveRequests: number = DEFAULT_MAX_LIVE_REQUESTS) {
    this.maxLiveRequests = maxLiveRequests;
    this.preventRequestsTill = performance.now();
  }

  update(urls: string[]): void {
    this.requested.clear();

    // Clear the hard refs:
    this.tiles.clear();

    // Hard ref anything requested that is cached:
    for (const url of urls) {
      this.requested.add(url);

      const tile = this.cache.get(url)?.deref();
      if (tile === undefined) {
        if (!this.live.has(url)) {
          this.queue.push(url);
        }
      } else {
        this.tiles.set(url, tile);
      }
    }

    this.queue = this.queue.filter(url => this.requested.has(url));
    for (const [url, abort] of this.aborts.entries()) {
      if (!this.requested.has(url)) {
        abort();
      }
    }

    this.processQueue();
  }

  get(url: string): TileData | undefined {
    return this.tiles.get(url);
  }

  private async requestTile(url: string): Promise<void> {
    this.live.add(url);

    try {
      const controller = new AbortController();
      this.aborts.set(url, () => controller.abort("panned away"));
      const signal = controller.signal;
      const tile = await fetchTile(url, signal);
      this.tiles.set(url, tile);
      this.cache.set(url, new WeakRef(tile));
    } catch (error) {
      if (!(error instanceof Error) || error.name !== "AbortError") {
        console.error("Failed to fetch tile:", error);
        this.preventRequestsTill = performance.now() + 10000;
      }
    } finally {
      this.live.delete(url);
      this.aborts.delete(url);
      this.processQueue();
    }
  }

  private processQueue(): void {
    if (performance.now() < this.preventRequestsTill) {
      return;
    }
    while (
      this.queue.length > 0 &&
      this.live.size < this.maxLiveRequests
    ) {
      const url = this.queue.shift()!;
      this.requestTile(url);
    }
  }
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

interface CompositorJob {
  // Unique identifier for this request:
  key: string;

  state: "pending"|"processing"|"done";

  // Details of the job:
  x: number;
  y: number;
  lod: number;
  repo: string;
  commit: string;
  kind: string;
  composite: string;

  // Required tiles:
  tiles: string[];

  // Final bitmap:
  bitmap?: ImageBitmap;
}

function compositeToRequiredTiles(r: CompositeTileRequest): string[] {
  const tiles: string[] = [];
  tiles.push(`/api/tile/${r.kind}/${r.repo}/${r.commit}/${r.lod}/${r.x}/${r.y}`);
  return tiles;
}

export class TileCompositor {
  private requested: Set<string> = new Set();
  private jobs: Map<string, CompositorJob> = new Map();
  private store: TileStore;

  constructor(store: TileStore) {
    this.store = store;
  }

  private toKey(request: CompositeTileRequest): string {
    return `${request.repo}_${request.commit}_${request.kind}_${request.composite}_${request.x}_${request.y}_${request.lod}`;
  }

  update(requests: CompositeTileRequest[]): void {
    this.requested.clear();

    for (const request of requests) {
      const key = this.toKey(request);
      this.requested.add(key);

      if (!this.jobs.has(key)) {
        this.jobs.set(key, {
          key,
          state: "pending",
          tiles: compositeToRequiredTiles(request),
          ...request
        });
      }
    }

    const jobsKeys = [...this.jobs.keys()];

    // Remove any dead jobs:
    for (const key of jobsKeys) {
      if (!this.requested.has(key)) {
        this.jobs.delete(key);
      }
    }

    // Request all tiles for active jobs:
    const tiles = [];
    for (const job of this.jobs.values()) {
      for (const tile of job.tiles) {
        tiles.push(tile);
      }
    }
    this.store.update(tiles);


    for (const job of this.jobs.values()) {
      if (job.state !== "pending") {
        continue;
      }

      let ready = true;
      for (const tile of job.tiles) {
        ready = ready && (this.store.get(tile) !== undefined);
      }

      if (!ready) {
        continue;
      }

      job.state = "processing";
      this.doJob(job);
    }
  }

  private async doJob(job: CompositorJob): Promise<void> {
    try {
      const data = this.store.get(job.tiles[0]!)!.data;
      const bitmap = await createTileBitmap(
          job.composite,
          data,
      )
      job.bitmap = bitmap;
    } catch (error) {
      console.error("Failed to create composite bitmap:", error);
    } finally {
      job.state = "done";
    }
  }

  get(request: CompositeTileRequest): ImageBitmap | undefined {
    const key = this.toKey(request);
    const job = this.jobs.get(key);
    if (job === undefined) {
      return undefined;
    }
    return job.bitmap;
  }
}
