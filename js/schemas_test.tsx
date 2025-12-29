import o from "ospec";
import {
  TILE_SIZE,
  GranularLineLengthSchema,
  IndexEntrySchema,
  IndexSchema,
  LineLengthSchema,
  FileSystemEntrySchema,
  InfoResponseSchema,
  RepoInfoSchema,
  RepoListResponseSchema,
} from "./schemas.js";

o.spec("schemas", () => {
  o.spec("constants", () => {
    o("TILE_SIZE is 64", () => {
      o(TILE_SIZE).equals(64);
    });
  });

  o.spec("GranularLineLengthSchema", () => {
    o("validates correct structure", () => {
      const valid = { LinesLengths: [10, 20, 30] };
      const result = GranularLineLengthSchema.safeParse(valid);
      o(result.success).equals(true);
    });

    o("accepts null LinesLengths", () => {
      const valid = { LinesLengths: null };
      const result = GranularLineLengthSchema.safeParse(valid);
      o(result.success).equals(true);
    });

    o("rejects invalid structure", () => {
      const invalid = { LinesLengths: "not an array" };
      const result = GranularLineLengthSchema.safeParse(invalid);
      o(result.success).equals(false);
    });
  });

  o.spec("IndexEntrySchema", () => {
    o("validates correct entry", () => {
      const valid = {
        path: "/some/path.ts",
        lineOffset: 100,
        lineCount: 50,
      };
      const result = IndexEntrySchema.safeParse(valid);
      o(result.success).equals(true);
    });

    o("rejects missing fields", () => {
      const invalid = { path: "/path.ts" };
      const result = IndexEntrySchema.safeParse(invalid);
      o(result.success).equals(false);
    });
  });

  o.spec("IndexSchema", () => {
    o("validates index with entries", () => {
      const valid = {
        entries: [
          { path: "/file1.ts", lineOffset: 0, lineCount: 100 },
          { path: "/file2.ts", lineOffset: 100, lineCount: 50 },
        ],
      };
      const result = IndexSchema.safeParse(valid);
      o(result.success).equals(true);
    });

    o("accepts null entries", () => {
      const valid = { entries: null };
      const result = IndexSchema.safeParse(valid);
      o(result.success).equals(true);
    });
  });

  o.spec("LineLengthSchema", () => {
    o("validates correct structure", () => {
      const valid = { maximum: 120 };
      const result = LineLengthSchema.safeParse(valid);
      o(result.success).equals(true);
    });

    o("rejects missing maximum", () => {
      const invalid = {};
      const result = LineLengthSchema.safeParse(invalid);
      o(result.success).equals(false);
    });
  });

  o.spec("FileSystemEntrySchema", () => {
    o("validates file entry", () => {
      const valid = {
        name: "test.ts",
        path: "/src/test.ts",
        type: "file",
        size: 1024,
        hash: "abc123",
      };
      const result = FileSystemEntrySchema.safeParse(valid);
      o(result.success).equals(true);
    });

    o("validates directory with children", () => {
      const valid = {
        name: "src",
        path: "/src",
        type: "directory",
        children: [
          {
            name: "file.ts",
            path: "/src/file.ts",
            type: "file",
            size: 512,
          },
        ],
      };
      const result = FileSystemEntrySchema.safeParse(valid);
      o(result.success).equals(true);
    });

    o("accepts optional fields", () => {
      const minimal = {
        name: "test",
        path: "/test",
        type: "file",
      };
      const result = FileSystemEntrySchema.safeParse(minimal);
      o(result.success).equals(true);
    });
  });

  o.spec("InfoResponseSchema", () => {
    o("validates response structure", () => {
      const valid = {
        entry: {
          name: "root",
          path: "/",
          type: "directory",
        },
      };
      const result = InfoResponseSchema.safeParse(valid);
      o(result.success).equals(true);
    });
  });

  o.spec("RepoInfoSchema", () => {
    o("validates repo info", () => {
      const valid = { name: "my-repo" };
      const result = RepoInfoSchema.safeParse(valid);
      o(result.success).equals(true);
    });
  });

  o.spec("RepoListResponseSchema", () => {
    o("validates repo list", () => {
      const valid = {
        repos: [{ name: "repo1" }, { name: "repo2" }],
      };
      const result = RepoListResponseSchema.safeParse(valid);
      o(result.success).equals(true);
    });

    o("accepts null repos", () => {
      const valid = { repos: null };
      const result = RepoListResponseSchema.safeParse(valid);
      o(result.success).equals(true);
    });
  });
});
