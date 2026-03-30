#!/usr/bin/env node

import { fileURLToPath } from "url";
import { dirname, resolve } from "path";
import { spawnSync } from "child_process";

// Директория, где находится фронтенд-пакет (а не проект, который вызывает npx)
const binDir = dirname(fileURLToPath(import.meta.url));
const packageDir = resolve(binDir, "..");

spawnSync("npm", ["run", "preview"], {
  stdio: "inherit",
  cwd: packageDir
});
