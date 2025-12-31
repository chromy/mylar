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
