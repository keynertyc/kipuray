// Regenerate after dependency changes: node scripts/licenses.mjs
import { execFileSync } from "node:child_process";
import { readFile, readdir, writeFile } from "node:fs/promises";
import path from "node:path";

const notices = [
  "Kipuray third-party license texts\nGenerated from the locked dependency graph.\n",
];
async function include(label, dir) {
  const names = await readdir(dir).catch(() => []);
  const licenses = names.filter((n) =>
    /^(licen[cs]e|copying|ofl)(\.|$)/i.test(n),
  );
  for (const name of licenses) {
    try {
      const text = await readFile(path.join(dir, name), "utf8");
      notices.push(
        `\n${"=".repeat(72)}\n${label} / ${name}\n${"=".repeat(72)}\n${text}`,
      );
    } catch {
      /* Some packages use a license directory instead of a text file. */
    }
  }
}
const lock = JSON.parse(await readFile("web/package-lock.json", "utf8"));
for (const [entry, pkg] of Object.entries(lock.packages)) {
  if (!entry || pkg.dev) continue;
  await include(
    `${entry.replace("node_modules/", "")} ${pkg.version}`,
    path.join("web", entry),
  );
}
const modules = execFileSync(
  "go",
  ["list", "-m", "-f", "{{.Path}}\t{{.Version}}\t{{.Dir}}", "all"],
  { encoding: "utf8" },
);
for (const line of modules.trim().split("\n")) {
  const [name, version, dir] = line.split("\t");
  if (name === "github.com/keynertyc/kipuray" || !dir) continue;
  await include(`${name} ${version}`, dir);
}
await writeFile("THIRD_PARTY_LICENSES.txt", notices.join("\n"));
