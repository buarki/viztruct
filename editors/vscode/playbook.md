# Viztruct VSCode extension — hacking runbook

## One-time setup

```bash
cd editors/vscode
npm install
```

This installs TypeScript and `@vscode/vsce` (the packager) into `node_modules/`.

You also need the `viztruct` Go binary reachable. Build it from the repo root:

```bash
cd ../..                 # back to the viztruct repo root
go install ./cmd/viztruct
```

The binary ends up at `$(go env GOPATH)/bin/viztruct`. If that's not on your `$PATH`, either add it, or set the extension's `viztruct.binaryPath` setting to the absolute path.

---

## Develop with live reload (F5 loop)

1. Open `editors/vscode/` as a workspace in VSCode (not the repo root — vsce and the debug config need this folder at the top).
2. Press **F5** (or Run → Start Debugging → "Run Extension"). A second VSCode window opens, titled `[Extension Development Host]`. Your extension is loaded in that window.
3. Edit `src/extension.ts` in the first window. The `watch` task auto-rebuilds into `dist/`.
4. In the Dev Host window: **Cmd+Shift+P → "Developer: Reload Window"** to pick up your changes.

Breakpoints in `src/extension.ts` work because sourcemaps are on (see `tsconfig.json`).

---

## Build an installable `.vsix`

```bash
cd editors/vscode
npm run package
```

Output: `viztruct-0.0.1.vsix` in the current directory.

What this does, step by step:

| Step | Command | Effect |
|---|---|---|
| 1 | `npm run package` | Runs the `package` script from `package.json` |
| 2 | → `vsce package` | The packager; reads `package.json` |
| 3 | → auto-runs `npm run vscode:prepublish` | vsce looks for this magic script name |
| 4 | → `npm run compile` | Which runs `tsc -p .` |
| 5 | → emits `dist/extension.js` | Sourcemaps and types are stripped from the final .vsix |
| 6 | `vsce` zips `package.json` + `dist/` | Produces the `.vsix` file |

No shell script needed — the whole chain is declared in `package.json` scripts.

---

## Install the `.vsix` into your regular VSCode

```bash
code --install-extension viztruct-0.0.1.vsix --force
```

`--force` replaces any previously installed version. Without it, install fails if the version number hasn't changed.

Restart VSCode if it's already running.

> If the `code` CLI isn't available: in VSCode, `Cmd+Shift+P` → "Shell Command: Install 'code' command in PATH".

---

## Uninstall

```bash
code --uninstall-extension buarki.viztruct
```

The extension ID is `<publisher>.<name>` — from the `publisher` and `name` fields in `package.json`.

---

## Verify what's installed

```bash
code --list-extensions --show-versions | grep viztruct
```

---

## Bump the version for a new build

Edit `version` in `package.json`, then re-run `npm run package`. You'll get a new `viztruct-<version>.vsix`.

---

## Settings the extension reads

| Setting | Default | Purpose |
|---|---|---|
| `viztruct.binaryPath` | `viztruct` | Path to the viztruct binary. Use an absolute path if `viztruct` isn't on the VSCode process's `$PATH` (common on macOS when launching VSCode from the Dock). |
| `viztruct.timeoutSeconds` | `30` | Per-invocation timeout. |

Set via `Cmd+,` → search `viztruct`, or directly in `settings.json`.
