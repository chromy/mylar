export interface LayerType {
  kind: string;
  composite: string;
  aggregation: string;
}

export const LAYER_OPTIONS: LayerType[] = [
  { kind: "length", composite: "direct", aggregation: "mean" },
  { kind: "indent", composite: "x10", aggregation: "max" },
  { kind: "offset", composite: "direct", aggregation: "mean" },
  { kind: "fileHash", composite: "hash", aggregation: "mode" },
  { kind: "fileExtension", composite: "hashRainbow", aggregation: "mode" },
];

export const DEFAULT_LAYER = LAYER_OPTIONS[0]!;

export const LAYER_LABELS: Record<string, string> = {
  length: "Line Length",
  indent: "Line Indent",
  offset: "Line Offset",
  fileHash: "File Hash",
  fileExtension: "File Type",
};
