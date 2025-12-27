import { z } from "zod";

export const TILE_SIZE = 128;

export const GranularLineLengthSchema = z.object({
  LinesLengths: z.number().array().nullable(),
});
export type GranularLineLength = z.infer<typeof GranularLineLengthSchema>;

export const IndexEntrySchema = z.object({
  path: z.string(),
  lineOffset: z.number(),
  lineCount: z.number(),
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

export type FileSystemEntry = {
  name: string;
  path: string;
  type: string;
  size?: number | undefined;
  hash?: string | undefined;
  children?: FileSystemEntry[] | undefined;
};
const FileSystemEntrySchemaShape = {
  name: z.string(),
  path: z.string(),
  type: z.string(),
  size: z.number().optional(),
  hash: z.string().optional(),
  children: z
    .lazy(() => FileSystemEntrySchema)
    .array()
    .optional(),
};
export const FileSystemEntrySchema: z.ZodType<FileSystemEntry> = z.object(
  FileSystemEntrySchemaShape,
);

export const InfoResponseSchema = z.object({
  entry: FileSystemEntrySchema,
});
export type InfoResponse = z.infer<typeof InfoResponseSchema>;

export const RepoInfoSchema = z.object({
  name: z.string(),
});
export type RepoInfo = z.infer<typeof RepoInfoSchema>;

export const RepoListResponseSchema = z.object({
  repos: RepoInfoSchema.array().nullable(),
});
export type RepoListResponse = z.infer<typeof RepoListResponseSchema>;
