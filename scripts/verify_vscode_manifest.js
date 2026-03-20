#!/usr/bin/env node
const fs = require('fs');
const path = require('path');

const repoRoot = path.resolve(__dirname, '..');
const packagePath = path.join(repoRoot, 'editors', 'vscode', 'package.json');
const pkg = JSON.parse(fs.readFileSync(packagePath, 'utf8'));

if (pkg.name !== 'kiro-vscode') {
  throw new Error(`unexpected package name: ${pkg.name}`);
}

if (!pkg.contributes?.configuration?.properties?.['kiro.lsp.path']) {
  throw new Error('missing kiro.lsp.path setting');
}

const repoUrl = pkg.repository?.url;
const homepage = pkg.homepage;
const bugsUrl = pkg.bugs?.url;
const expectedRepoUrl = 'https://github.com/tonitienda/kiro-lang.git';
const expectedHomepage = 'https://github.com/tonitienda/kiro-lang';
const expectedBugsUrl = 'https://github.com/tonitienda/kiro-lang/issues';

if (repoUrl !== expectedRepoUrl) {
  throw new Error(`unexpected repository url: ${repoUrl}`);
}

if (homepage !== expectedHomepage) {
  throw new Error(`unexpected homepage url: ${homepage}`);
}

if (bugsUrl !== expectedBugsUrl) {
  throw new Error(`unexpected bugs url: ${bugsUrl}`);
}

console.log('manifest ok');
