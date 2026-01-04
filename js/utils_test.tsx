import o from "ospec";
import {
  lodToSize,
  getGridSide,
  lineToWorld,
  worldToTile,
  tileToWorld,
  worldToLine,
} from "./utils.js";
import type {
  TileLayout,
  WorldPosition,
  TilePosition,
  LinePosition,
} from "./utils.js";

o.spec("utils", () => {
  o.spec("lodToSize", () => {
    o("converts LOD to correct size", () => {
      o(lodToSize(0)).equals(64);
      o(lodToSize(1)).equals(128);
      o(lodToSize(2)).equals(256);
      o(lodToSize(3)).equals(512);
    });
  });

  o.spec("getGridSide", () => {
    o("handles edge cases", () => {
      o(getGridSide({ lineCount: 0 })).equals(1);
      o(getGridSide({ lineCount: 1 })).equals(1);
    });

    o("calculates correct grid side", () => {
      o(getGridSide({ lineCount: 4 })).equals(2);
      o(getGridSide({ lineCount: 16 })).equals(4);
      o(getGridSide({ lineCount: 64 })).equals(8);
    });
  });

  o.spec("Coordinate conversions", () => {
    const layout: TileLayout = { lineCount: 1000 };

    o("lineToWorld/worldToLine round trip", () => {
      const testLines = [0, 1, 10, 100];

      testLines.forEach(line => {
        const world = lineToWorld(line, layout);
        const backToLine = worldToLine(world, layout);
        o(backToLine).equals(line);
      });
    });

    o("worldToTile converts correctly", () => {
      const world: WorldPosition = { x: 130, y: 200 };
      const tile = worldToTile(world, layout);

      o(tile.lod).equals(0);
      o(tile.tileX).equals(2); // 130 / 64 = 2.03... -> 2
      o(tile.tileY).equals(3); // 200 / 64 = 3.125 -> 3
      o(tile.offsetX).equals(2); // 130 % 64 = 2
      o(tile.offsetY).equals(8); // 200 % 64 = 8
    });

    o("tileToWorld converts correctly", () => {
      const tile: TilePosition = {
        lod: 0,
        tileX: 2,
        tileY: 3,
        offsetX: 2,
        offsetY: 8,
      };
      const world = tileToWorld(tile, layout);

      o(world.x).equals(130); // 2 * 64 + 2 = 130
      o(world.y).equals(200); // 3 * 64 + 8 = 200
    });
  });
});
