import { delimiter, isAbsolute, join } from "node:path";

const binaryName = process.platform === "win32" ? "chik.exe" : "chik";

// Picks the chik binary to start. A configured path wins; otherwise the first
// directory on PATH that holds a file named chik is used. Returning undefined
// lets the caller report one clear error instead of a spawn failure.
export function resolveServer(
  configured: string,
  searchPath: string,
  exists: (path: string) => boolean,
): string | undefined {
  const trimmed = configured.trim();
  if (trimmed.length > 0) {
    return isAbsolute(trimmed) && !exists(trimmed) ? undefined : trimmed;
  }
  for (const directory of searchPath.split(delimiter)) {
    if (directory.length === 0) continue;
    const candidate = join(directory, binaryName);
    if (exists(candidate)) return candidate;
  }
  return undefined;
}
