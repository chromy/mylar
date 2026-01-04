export interface LayerType {
  kind: string;
  composite: string;
  aggregation: string;
}

export const LAYER_OPTIONS: LayerType[] = [
  { kind: "length", composite: "255|swap|sub|dup|dup", aggregation: "mean" },
  {
    kind: "indent",
    composite: "10|mul|0|max|255|min|255|swap|sub|dup|dup",
    aggregation: "max",
  },
  { kind: "offset", composite: "255|swap|sub|dup|dup", aggregation: "mean" },
  {
    kind: "fileHash",
    composite: "1|swap|1|swap|hash|360|mod|oklchToSrgb|toByteX3",
    aggregation: "mode",
  },
  {
    kind: "fileExtension",
    composite: "hash|int32ToUnit|rainbow|oklchToSrgb|toByteX3",
    aggregation: "mode",
  },
];

export const DEFAULT_LAYER = LAYER_OPTIONS[0]!;

export const LAYER_LABELS: Record<string, string> = {
  length: "Line Length",
  indent: "Line Indent",
  offset: "Line Offset",
  fileHash: "File Hash",
  fileExtension: "File Type",
};
