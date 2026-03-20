#!/usr/bin/env node
const fs = require('fs');
const path = require('path');

const repoRoot = path.resolve(__dirname, '..');
const extensionPath = path.join(repoRoot, 'editors', 'vscode', 'extension.js');
const src = fs.readFileSync(extensionPath, 'utf8');

if (!/command: 'kiro'/.test(src) || !/args: \['lsp'\]/.test(src)) {
  throw new Error('default kiro lsp entrypoint missing');
}

console.log('lsp entrypoint ok');
