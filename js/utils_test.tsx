import o from "ospec";
import { lodToSize } from "./utils.js";

o.spec("utils", () => {
  o.spec("lodToSize", () => {
    o("converts LOD to correct size", () => {
      o(lodToSize(0)).equals(64);
      o(lodToSize(1)).equals(128);
      o(lodToSize(2)).equals(256);
      o(lodToSize(3)).equals(512);
    });
  });
});
