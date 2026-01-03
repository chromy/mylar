import { z } from "zod";

export const TILE_SIZE = 64;

export const TileMetadataSchema = z.object({
  x: z.number(),
  y: z.number(),
  lod: z.number(),
});
export type TileMetadata = z.infer<typeof TileMetadataSchema>;

export const IndexEntrySchema = z.object({
  path: z.string(),
  lineOffset: z.number(),
  lineCount: z.number(),
  hash: z.number().array().length(20),
});
export type IndexEntry = z.infer<typeof IndexEntrySchema>;

export const WorldPositionSchema = z.object({
  X: z.number(),
  Y: z.number(),
});
export type WorldPosition = z.infer<typeof WorldPositionSchema>;

export const TilePositionSchema = z.object({
  Lod: z.number(),
  TileX: z.number(),
  TileY: z.number(),
  OffsetX: z.number(),
  OffsetY: z.number(),
});
export type TilePosition = z.infer<typeof TilePositionSchema>;

export const FileByLineResponseSchema = z.object({
  entry: IndexEntrySchema,
  content: z.string(),
  lineOffset: z.number(),
  worldPosition: WorldPositionSchema,
  tilePosition: TilePositionSchema,
});
export type FileByLineResponse = z.infer<typeof FileByLineResponseSchema>;

export const IndexSchema = z.object({
  entries: IndexEntrySchema.array().nullable(),
});
export type Index = z.infer<typeof IndexSchema>;

export const LineLengthSchema = z.object({
  maximum: z.number(),
});
export type LineLength = z.infer<typeof LineLengthSchema>;

export const RepoInfoSchema = z.object({
  id: z.string(),
  owner: z.string().optional(),
  name: z.string().optional(),
});
export type RepoInfo = z.infer<typeof RepoInfoSchema>;

export const RepoListResponseSchema = z.object({
  repos: RepoInfoSchema.array().nullable(),
});
export type RepoListResponse = z.infer<typeof RepoListResponseSchema>;

export const ResolveCommittishResponseSchema = z.object({
  commit: z.string(),
  tree: z.string(),
});
export type ResolveCommittishResponse = z.infer<
  typeof ResolveCommittishResponseSchema
>;

export const TreeEntrySchema = z.object({
  name: z.string(),
  hash: z.string(),
  mode: z.number(),
});
export type TreeEntry = z.infer<typeof TreeEntrySchema>;

export const TreeEntriesSchema = z.object({
  entries: TreeEntrySchema.array().nullable(),
});
export type TreeEntries = z.infer<typeof TreeEntriesSchema>;

export const MemoryStatsSchema = z.object({
  alloc: z.number(),
  total_alloc: z.number(),
  sys: z.number(),
  num_gc: z.number(),
  heap_alloc: z.number(),
  heap_sys: z.number(),
  heap_inuse: z.number(),
  heap_released: z.number(),
  stack_inuse: z.number(),
  stack_sys: z.number(),
});
export type MemoryStats = z.infer<typeof MemoryStatsSchema>;

export const VarzResponseSchema = z.object({
  version: z.string(),
  build_time: z.string(),
  go_version: z.string(),
  start_time: z.coerce.date(),
  uptime: z.string(),
  memory: MemoryStatsSchema,
});
export type VarzResponse = z.infer<typeof VarzResponseSchema>;
