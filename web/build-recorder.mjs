import { copyFile } from "node:fs/promises";
import { build } from "esbuild";
await build({
  entryPoints: ["src/recorder.ts"],
  bundle: true,
  format: "iife",
  target: "chrome100",
  outfile: "../internal/browser/recorder.js",
  minify: true,
});

await copyFile(
  "../THIRD_PARTY_LICENSES.txt",
  "../internal/web/dist/THIRD_PARTY_LICENSES.txt",
);
