import o from "ospec";
import {
  TILE_SIZE,
  IndexEntrySchema,
  IndexSchema,
  LineLengthSchema,
  RepoInfoSchema,
  RepoListResponseSchema,
} from "./schemas.js";

o.spec("schemas", () => {
  o.spec("constants", () => {
    o("TILE_SIZE is 64", () => {
      o(TILE_SIZE).equals(64);
    });
  });

  o.spec("IndexEntrySchema", () => {
    o("validates correct entry", () => {
      const valid = {
        path: "/some/path.ts",
        lineOffset: 100,
        lineCount: 50,
        hash: [
          1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
        ],
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
          {
            path: "/file1.ts",
            lineOffset: 0,
            lineCount: 100,
            hash: [
              1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19,
              20,
            ],
          },
          {
            path: "/file2.ts",
            lineOffset: 100,
            lineCount: 50,
            hash: [
              1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19,
              20,
            ],
          },
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
