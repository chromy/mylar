import o from "ospec";
import {
  lodToSize,
  initialSize,
  lineToWorld,
  worldToTile,
  tileToWorld,
  worldToLine,
  mortonEncode,
  mortonDecode,
  spreadBits,
  compactBits,
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

  o.spec("initialSize", () => {
    o("handles edge cases", () => {
      o(initialSize(0)).equals(2);
      o(initialSize(1)).equals(2);
    });

    o("calculates correct initial size", () => {
      o(initialSize(4)).equals(2);
      o(initialSize(16)).equals(4);
      o(initialSize(64)).equals(8);
    });
  });

  o.spec("Morton encoding/decoding", () => {
    o("spreadBits spreads bits correctly", () => {
      o(spreadBits(0)).equals(0);
      o(spreadBits(1)).equals(1);
      o(spreadBits(2)).equals(4);
      o(spreadBits(3)).equals(5);
    });

    o("compactBits compacts bits correctly", () => {
      o(compactBits(0)).equals(0);
      o(compactBits(1)).equals(1);
      o(compactBits(4)).equals(2);
      o(compactBits(5)).equals(3);
    });

    o("mortonEncode/mortonDecode round trip", () => {
      const testCases: [number, number][] = [
        [0, 0],
        [1, 1],
        [2, 3],
        [10, 20],
      ];

      testCases.forEach(([x, y]) => {
        const encoded = mortonEncode(x, y);
        const [decodedX, decodedY] = mortonDecode(encoded);
        o(decodedX).equals(x);
        o(decodedY).equals(y);
      });
    });
  });

  o.spec("Coordinate conversions", () => {
    const layout: TileLayout = { LastLine: 1000 };

    o("lineToWorld/worldToLine round trip", () => {
      const testLines = [0, 1, 10, 100];

      testLines.forEach(line => {
        const world = lineToWorld(line, layout);
        const backToLine = worldToLine(world, layout);
        o(backToLine).equals(line);
      });
    });

    o("worldToTile converts correctly", () => {
      const world: WorldPosition = { X: 130, Y: 200 };
      const tile = worldToTile(world, layout);

      o(tile.Lod).equals(0);
      o(tile.TileX).equals(2); // 130 / 64 = 2.03... -> 2
      o(tile.TileY).equals(3); // 200 / 64 = 3.125 -> 3
      o(tile.OffsetX).equals(2); // 130 % 64 = 2
      o(tile.OffsetY).equals(8); // 200 % 64 = 8
    });

    o("tileToWorld converts correctly", () => {
      const tile: TilePosition = {
        Lod: 0,
        TileX: 2,
        TileY: 3,
        OffsetX: 2,
        OffsetY: 8,
      };
      const world = tileToWorld(tile, layout);

      o(world.X).equals(130); // 2 * 64 + 2 = 130
      o(world.Y).equals(200); // 3 * 64 + 8 = 200
    });
  });
});
