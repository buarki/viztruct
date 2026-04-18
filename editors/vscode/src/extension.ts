import * as vscode from 'vscode';
import * as path from 'path';
import * as fs from 'fs';
import { execFile } from 'child_process';
import { promisify } from 'util';

const execFileAsync = promisify(execFile);

const STRUCT_RE = /^type\s+(\w+)(?:\[[^\]]+\])?\s+struct\s*\{/gm;

interface Field {
  name: string;
  type: string;
  offset: number;
  size: number;
  align: number;
  is_padding: boolean;
}

interface Info {
  name: string;
  original_size: number;
  optimized_size: number;
  wasted_bytes: number;
  wasted_percent: number;
  fields: Field[];
  optimized_fields: Field[];
}

class StructCodeLensProvider implements vscode.CodeLensProvider {
  provideCodeLenses(document: vscode.TextDocument): vscode.CodeLens[] {
    const lenses: vscode.CodeLens[] = [];
    const text = document.getText();
    for (const match of text.matchAll(STRUCT_RE)) {
      if (match.index === undefined) continue;
      const structName = match[1];
      const pos = document.positionAt(match.index);
      const range = new vscode.Range(pos, pos);
      lenses.push(
        new vscode.CodeLens(range, {
          title: '$(symbol-struct) Analyze layout',
          command: 'viztruct.analyzeStruct',
          arguments: [document.uri, structName],
        }),
      );
    }
    return lenses;
  }
}

function flattenInfos(parsed: unknown): Info[] {
  if (!Array.isArray(parsed)) return [];
  const out: Info[] = [];
  for (const item of parsed) {
    if (Array.isArray(item)) out.push(...(item as Info[]));
    else if (item && typeof item === 'object') out.push(item as Info);
  }
  return out;
}

let panel: vscode.WebviewPanel | undefined;
let currentStruct: { uri: vscode.Uri; name: string } | undefined;
let extensionRoot: string | undefined;

function bundledBinaryName(platform: NodeJS.Platform, arch: string): string | undefined {
  const osMap: Partial<Record<NodeJS.Platform, string>> = {
    darwin: 'darwin',
    linux: 'linux',
    win32: 'windows',
  };
  const archMap: Record<string, string> = { x64: 'amd64', arm64: 'arm64' };
  const os = osMap[platform];
  const a = archMap[arch];
  if (!os || !a) return undefined;
  const ext = os === 'windows' ? '.exe' : '';
  return `viztruct-${os}-${a}${ext}`;
}

function resolveBinaryPath(): string {
  const setting = vscode.workspace.getConfiguration('viztruct').get<string>('binaryPath', '').trim();
  if (setting) return setting;

  if (extensionRoot) {
    const name = bundledBinaryName(process.platform, process.arch);
    if (name) {
      const bundled = path.join(extensionRoot, 'bin', name);
      if (fs.existsSync(bundled)) {
        // .vsix extraction may drop the exec bit on POSIX; ensure it's runnable.
        if (process.platform !== 'win32') {
          try { fs.chmodSync(bundled, 0o755); } catch { /* non-fatal */ }
        }
        return bundled;
      }
    }
  }
  return 'viztruct';
}

async function analyzeStruct(uri: vscode.Uri, structName: string): Promise<void> {
  if (uri.scheme !== 'file') {
    vscode.window.showErrorMessage('Viztruct only runs on file-backed Go sources.');
    return;
  }

  const pkgDir = path.dirname(uri.fsPath);
  const binaryPath = resolveBinaryPath();
  const timeoutSeconds = vscode.workspace.getConfiguration('viztruct').get<number>('timeoutSeconds', 30);

  const args = [
    '--path', pkgDir,
    '--format', 'json',
    '--skip-errors',
    '--timeout', String(timeoutSeconds),
    '--struct', structName,
  ];

  let stdout: string;
  try {
    const result = await vscode.window.withProgress(
      { location: vscode.ProgressLocation.Notification, title: `Viztruct: analyzing ${structName}…` },
      () =>
        execFileAsync(binaryPath, args, {
          maxBuffer: 16 * 1024 * 1024,
          timeout: (timeoutSeconds + 5) * 1000,
        }),
    );
    stdout = result.stdout;
  } catch (err) {
    const e = err as NodeJS.ErrnoException & {
      stderr?: string;
      stdout?: string;
      code?: string | number;
      signal?: string;
    };
    console.error('[viztruct] subprocess failed', { binaryPath, args, error: e });
    const parts: string[] = [];
    if (e.code === 'ENOENT') {
      parts.push(`binary not found: "${binaryPath}" (set "viztruct.binaryPath" or install viztruct on PATH)`);
    } else if (e.signal) {
      parts.push(`killed by signal ${e.signal}`);
    } else if (typeof e.code === 'number') {
      parts.push(`exit code ${e.code}`);
    }
    const stderrTail = (e.stderr || '').split('\n').filter((l) => l.trim()).slice(-3).join(' | ');
    const stdoutTail = (e.stdout || '').split('\n').filter((l) => l.trim()).slice(-2).join(' | ');
    if (stderrTail) parts.push(`stderr: ${stderrTail}`);
    else if (stdoutTail) parts.push(`stdout: ${stdoutTail}`);
    else if (e.message) parts.push(e.message.split('\n')[0]);
    vscode.window.showErrorMessage(`Viztruct failed — ${parts.join(' — ') || String(err)}`);
    return;
  }

  let parsed: unknown;
  try {
    parsed = JSON.parse(stdout);
  } catch {
    vscode.window.showErrorMessage('Viztruct returned invalid JSON.');
    return;
  }

  const infos: Info[] = flattenInfos(parsed);
  const info = infos.find((i) => i.name === structName);
  if (!info) {
    vscode.window.showWarningMessage(
      `Viztruct did not return layout data for "${structName}". The struct may be in an unbuildable package.`,
    );
    return;
  }

  showPanel(info, uri);
}

function showPanel(info: Info, uri: vscode.Uri): void {
  currentStruct = { uri, name: info.name };
  if (!panel) {
    panel = vscode.window.createWebviewPanel(
      'viztruct',
      `Viztruct: ${info.name}`,
      vscode.ViewColumn.Beside,
      { retainContextWhenHidden: true, enableScripts: true },
    );
    panel.onDidDispose(() => {
      panel = undefined;
      currentStruct = undefined;
    });
    panel.webview.onDidReceiveMessage(async (msg: { type?: string }) => {
      if (msg?.type === 'apply' && currentStruct) {
        await applyRewrite(currentStruct.uri, currentStruct.name);
      }
    });
  }
  panel.title = `Viztruct: ${info.name}`;
  panel.webview.html = renderHtml(info, uri);
  panel.reveal(vscode.ViewColumn.Beside, true);
}

async function applyRewrite(uri: vscode.Uri, structName: string): Promise<void> {
  const doc = await vscode.workspace.openTextDocument(uri);
  if (doc.isDirty) {
    const save = 'Save and rewrite';
    const pick = await vscode.window.showWarningMessage(
      `${path.basename(uri.fsPath)} has unsaved changes. Save before rewriting ${structName}?`,
      { modal: true },
      save,
    );
    if (pick !== save) return;
    await doc.save();
  }

  const binaryPath = resolveBinaryPath();
  const timeoutSeconds = vscode.workspace.getConfiguration('viztruct').get<number>('timeoutSeconds', 30);
  const args = ['--rewrite', '--file', uri.fsPath, '--struct', structName];

  let newSource: string;
  try {
    const result = await vscode.window.withProgress(
      { location: vscode.ProgressLocation.Notification, title: `Viztruct: rewriting ${structName}…` },
      () =>
        execFileAsync(binaryPath, args, {
          maxBuffer: 32 * 1024 * 1024,
          timeout: (timeoutSeconds + 5) * 1000,
        }),
    );
    newSource = result.stdout;
  } catch (err) {
    const e = err as Omit<NodeJS.ErrnoException, 'code'> & {
      stderr?: string;
      code?: number | string;
    };
    console.error('[viztruct] rewrite failed', { binaryPath, args, error: e });
    const reason = (e.stderr || '').trim().split('\n').slice(-3).join(' | ') || e.message || String(err);
    if (e.code === 2) {
      vscode.window.showWarningMessage(`Viztruct: ${reason}`);
    } else {
      vscode.window.showErrorMessage(`Viztruct rewrite failed — ${reason}`);
    }
    panel?.webview.postMessage({ type: 'applyDone', ok: false });
    return;
  }

  if (!newSource) {
    vscode.window.showErrorMessage('Viztruct rewrite returned empty output.');
    panel?.webview.postMessage({ type: 'applyDone', ok: false });
    return;
  }

  const fullRange = new vscode.Range(
    doc.positionAt(0),
    doc.positionAt(doc.getText().length),
  );
  const edit = new vscode.WorkspaceEdit();
  edit.replace(uri, fullRange, newSource);
  const ok = await vscode.workspace.applyEdit(edit);
  if (!ok) {
    vscode.window.showErrorMessage('Failed to apply rewrite to document.');
    panel?.webview.postMessage({ type: 'applyDone', ok: false });
    return;
  }

  panel?.webview.postMessage({ type: 'applyDone', ok: true });

  const saved = await doc.save();
  if (!saved) {
    vscode.window.showWarningMessage(
      `Rewrote ${structName}, but the file could not be saved. Review the diff and save manually.`,
    );
    return;
  }

  vscode.window.showInformationMessage(
    `Rewrote and saved ${structName}. Cmd+Z in the editor to undo.`,
  );
  await analyzeStruct(uri, structName);
}

function escapeHtml(s: string | undefined | null): string {
  if (s === undefined || s === null) return '';
  return String(s).replace(/[<>&"']/g, (c) => {
    switch (c) {
      case '<': return '&lt;';
      case '>': return '&gt;';
      case '&': return '&amp;';
      case '"': return '&quot;';
      case "'": return '&#39;';
      default: return c;
    }
  });
}

function renderFieldRow(f: Field): string {
  const cls = f.is_padding ? 'padding' : '';
  const name = f.is_padding ? '⟨padding⟩' : escapeHtml(f.name);
  const typeCell = f.is_padding ? '<span class="muted">—</span>' : `<code>${escapeHtml(f.type)}</code>`;
  return `
    <tr class="${cls}">
      <td>${name}</td>
      <td>${typeCell}</td>
      <td class="num">${f.offset}</td>
      <td class="num">${f.size}</td>
      <td class="num">${f.align}</td>
    </tr>`;
}

function renderTable(fields: Field[]): string {
  return `
    <table>
      <thead>
        <tr><th>Field</th><th>Type</th><th>Offset</th><th>Size</th><th>Align</th></tr>
      </thead>
      <tbody>${fields.map(renderFieldRow).join('')}</tbody>
    </table>`;
}

function renderHtml(info: Info, uri: vscode.Uri): string {
  const savings = info.original_size - info.optimized_size;
  const pct = info.wasted_percent.toFixed(1);
  const rel = vscode.workspace.asRelativePath(uri);
  const canRewrite = savings > 0;

  const summary = canRewrite
    ? `
    <span class="badge">${info.original_size} B</span> current
    →
    <span class="badge">${info.optimized_size} B</span> optimized
    <span class="badge win">saves ${savings} B</span>
    <span class="badge">${pct}% wasted</span>`
    : `<span class="badge">${info.original_size} B</span>`;

  const layout = canRewrite
    ? `
  <div class="panels">
    <div>
      <h2>Current layout</h2>
      ${renderTable(info.fields)}
    </div>
    <div>
      <h2>Optimized layout</h2>
      ${renderTable(info.optimized_fields)}
    </div>
  </div>`
    : `
  <div class="single-panel">
    <h2>Layout</h2>
    ${renderTable(info.fields)}
  </div>`;

  return `<!doctype html>
<html>
<head>
<meta charset="utf-8" />
<style>
  body {
    font-family: var(--vscode-font-family);
    color: var(--vscode-foreground);
    padding: 1rem;
    margin: 0;
  }
  h1 { font-size: 1.25rem; margin: 0 0 0.25rem 0; }
  .meta { color: var(--vscode-descriptionForeground); font-size: 0.85rem; margin-bottom: 1rem; }
  .summary { margin-bottom: 1.25rem; font-size: 0.95rem; }
  .badge {
    display: inline-block;
    padding: 2px 8px;
    border-radius: 4px;
    background: var(--vscode-badge-background);
    color: var(--vscode-badge-foreground);
    font-size: 0.85rem;
    margin: 0 2px;
  }
  .badge.win { background: var(--vscode-testing-iconPassed, #89d185); color: var(--vscode-editor-background); }
  .panels { display: grid; grid-template-columns: 1fr 1fr; gap: 1.5rem; }
  .single-panel { max-width: 560px; }
  h2 { font-size: 0.95rem; margin: 0 0 0.5rem 0; text-transform: uppercase; letter-spacing: 0.05em; color: var(--vscode-descriptionForeground); }
  table { width: 100%; border-collapse: collapse; font-size: 0.85rem; }
  th, td { padding: 4px 8px; text-align: left; border-bottom: 1px solid var(--vscode-panel-border); }
  th { font-weight: 600; color: var(--vscode-descriptionForeground); }
  .num { text-align: right; font-variant-numeric: tabular-nums; }
  tr.padding { opacity: 0.5; font-style: italic; }
  .muted { color: var(--vscode-descriptionForeground); }
  code { font-family: var(--vscode-editor-font-family); font-size: 0.85rem; }
  .actions { margin-top: 1.5rem; }
  button {
    background: var(--vscode-button-background);
    color: var(--vscode-button-foreground);
    border: none;
    padding: 6px 14px;
    font-size: 0.85rem;
    border-radius: 2px;
  }
  button[disabled] { cursor: not-allowed; opacity: 0.5; }
  button:not([disabled]) { cursor: pointer; }
  button:not([disabled]):hover { background: var(--vscode-button-hoverBackground); }
  .hint { margin-left: 0.75rem; color: var(--vscode-descriptionForeground); font-size: 0.8rem; }
  .optimal { color: var(--vscode-testing-iconPassed, #89d185); font-size: 0.9rem; font-weight: 500; }
</style>
</head>
<body>
  <h1>${escapeHtml(info.name)}</h1>
  <div class="meta">${escapeHtml(rel)}</div>
  <div class="summary">${summary}
  </div>
  ${layout}
  <div class="actions">
    ${canRewrite
      ? `<button id="applyBtn">Apply optimization</button>
         <span class="hint">Applies as an editor edit — review the diff and save to confirm.</span>`
      : `<div class="optimal">✓ Already optimally laid out — no action needed</div>`}
  </div>
  <script>
    const vscode = acquireVsCodeApi();
    const btn = document.getElementById('applyBtn');
    const originalLabel = btn ? btn.textContent : '';
    if (btn && !btn.disabled) {
      btn.addEventListener('click', () => {
        btn.disabled = true;
        btn.textContent = 'Rewriting…';
        vscode.postMessage({ type: 'apply' });
      });
    }
    window.addEventListener('message', (event) => {
      const msg = event.data;
      if (!msg || msg.type !== 'applyDone' || !btn) return;
      if (msg.ok) {
        btn.textContent = 'Applied — re-analyze to refresh';
        btn.disabled = true;
      } else {
        btn.textContent = originalLabel;
        btn.disabled = false;
      }
    });
  </script>
</body>
</html>`;
}

export function activate(context: vscode.ExtensionContext): void {
  extensionRoot = context.extensionPath;
  context.subscriptions.push(
    vscode.languages.registerCodeLensProvider(
      { language: 'go', scheme: 'file' },
      new StructCodeLensProvider(),
    ),
    vscode.commands.registerCommand('viztruct.analyzeStruct', analyzeStruct),
  );
}

export function deactivate(): void {
  panel?.dispose();
  panel = undefined;
}
