import { z } from "zod";

export const TILE_SIZE = 64;

export const GranularLineLengthSchema = z.object({
  LinesLengths: z.number().array().nullable(),
});
export type GranularLineLength = z.infer<typeof GranularLineLengthSchema>;

export const IndexEntrySchema = z.object({
  path: z.string(),
  lineOffset: z.number(),
  lineCount: z.number(),
  hash: z.number().array().length(20),
});
export type IndexEntry = z.infer<typeof IndexEntrySchema>;

export const IndexSchema = z.object({
  entries: IndexEntrySchema.array().nullable(),
});
export type Index = z.infer<typeof IndexSchema>;

export const LineLengthSchema = z.object({
  maximum: z.number(),
});
export type LineLength = z.infer<typeof LineLengthSchema>;

export const RepoInfoSchema = z.object({
  name: z.string(),
});
export type RepoInfo = z.infer<typeof RepoInfoSchema>;

export const RepoListResponseSchema = z.object({
  repos: RepoInfoSchema.array().nullable(),
});
export type RepoListResponse = z.infer<typeof RepoListResponseSchema>;

export const TreeEntrySchema = z.object({
  name: z.string(),
  mode: z.string(),
  hash: z.number().array().length(20),
});
export type TreeEntry = z.infer<typeof TreeEntrySchema>;

export const TreeEntriesSchema = z.object({
  entries: TreeEntrySchema.array().nullable(),
});
export type TreeEntries = z.infer<typeof TreeEntriesSchema>;

export const TileMetadataSchema = z.object({
  y: z.number(),
  x: z.number(),
  lod: z.number(),
});
export type TileMetadata = z.infer<typeof TileMetadataSchema>;
