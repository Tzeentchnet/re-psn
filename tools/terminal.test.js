// Tests for site/terminal.js (the pure part: run and complete). Run from the repository root:
//   node --test tools/terminal.test.js
// They live in tools/, not site/, because the Pages workflow publishes everything under site/.
'use strict';
const test = require('node:test');
const assert = require('node:assert');
const fs = require('node:fs');
const path = require('node:path');

const root = path.join(__dirname, '..');
const T = require(path.join(root, 'site', 'terminal.js'));
const data = JSON.parse(fs.readFileSync(path.join(root, 'site', 'data.json'), 'utf8'));
const source = fs.readFileSync(path.join(root, 'site', 'terminal.js'), 'utf8');

function text(res) {
  return res.lines.map((l) => (Array.isArray(l) ? l : [l]).map((s) => (typeof s === 'string' ? s : s.t)).join('')).join('\n');
}
function links(res) {
  const out = [];
  res.lines.forEach((l) => (Array.isArray(l) ? l : [l]).forEach((s) => { if (typeof s === 'object' && s.href) out.push(s.href); }));
  return out;
}

const COMMANDS = ['help', '--help', 'features', 'roadmap', 'settings', 'install', 'connect', 'version', 'changelog', 'about',
  'credits', 'licence', 'license', 'docs', 'links', 'theme', 'clear', 'history'];

test('every command answers and links stay on github.com or in the repository', () => {
  for (const c of COMMANDS) {
    const res = T.run(c, data, { history: ['x'], theme: 'system' });
    assert.ok(!res.error, c);
    if (c !== 'clear') assert.ok(res.lines.length > 0, c);
    for (const h of links(res)) assert.match(h, /^(https:\/\/github\.com\/|#)/, c + ' ' + h);
  }
});

test('unknown input', () => {
  assert.strictEqual(text(T.run('frobnicate now', data)), 'command not found: frobnicate, type --help for options');
  assert.deepStrictEqual(T.run('   ', data).lines, []);
});

test('help lists every command', () => {
  const t = text(T.run('--help', data));
  for (const c of ['features', 'roadmap', 'settings', 'install', 'connect', 'version', 'changelog', 'about', 'credits', 'licence', 'docs', 'links', 'theme', 'clear', 'history']) {
    assert.ok(t.includes(c), c);
  }
});

test('features: filter, detail, errors', () => {
  const all = text(T.run('features', data));
  assert.ok(all.startsWith(data.features.length + ' features'));
  const planned = text(T.run('features --status planned', data));
  assert.ok(planned.includes(data.roadmap.Next[0]));
  assert.ok(!planned.includes('[available]'));
  const id = data.features[0].id;
  const one = text(T.run('features ' + id, data));
  assert.ok(one.includes('Evidence: ' + data.features[0].evidence));
  assert.match(text(T.run('features --status nonsense', data)), /unknown status nonsense/);
  assert.match(text(T.run('features nope', data)), /no feature with id nope/);
  assert.match(text(T.run('features --status', data)), /needs a value/);
});

test('settings: list and one by flag or variable', () => {
  assert.ok(text(T.run('settings', data)).includes('RE_PSN_ADDR'));
  assert.ok(text(T.run('settings addr', data)).includes('-addr'));
  assert.ok(text(T.run('settings RE_PSN_ADDR', data)).includes('-addr'));
  assert.match(text(T.run('settings zzz', data)), /no setting named zzz/);
});

test('version, roadmap, install, connect use the data', () => {
  assert.ok(text(T.run('version', data)).includes('v' + data.version));
  const r = text(T.run('roadmap', data));
  assert.ok(r.includes('# Done (' + data.roadmap.Done.length + ')'));
  assert.ok(text(T.run('install', data)).includes(data.install.release_tag));
  assert.ok(text(T.run('connect', data)).includes(data.connect.intro));
});

test('theme and history and clear', () => {
  assert.strictEqual(T.run('theme light', data).theme, 'light');
  assert.match(text(T.run('theme', data, { theme: 'dark' })), /theme: dark/);
  assert.match(text(T.run('theme pink', data)), /unknown value pink/);
  assert.match(text(T.run('history', data, { history: ['features', 'roadmap'] })), /1 {2}features\n2 {2}roadmap/);
  assert.strictEqual(T.run('clear', data).clear, true);
});

test('links in text are limited to an allowlist', () => {
  assert.strictEqual(T.hrefFor('https://evil.example/x', data), null);
  assert.strictEqual(T.hrefFor('javascript:alert(1)', data), null);
  assert.strictEqual(T.hrefFor('//evil.example', data), null);
  assert.strictEqual(T.hrefFor('docs/api.md', data), data.repository + '/blob/main/docs/api.md');
  assert.deepStrictEqual(T.md('see [x](http://a.example) and `y`', data), ['see ', 'x', ' and ', { t: 'y', c: 'code' }]);
});

test('tab completion', () => {
  assert.strictEqual(T.complete('fea', data).value, 'features ');
  assert.strictEqual(T.complete('features --st', data).value, 'features --status ');
  assert.strictEqual(T.complete('features --status pl', data).value, 'features --status planned ');
  assert.strictEqual(T.complete('theme d', data).value, 'theme dark ');
  assert.ok(T.complete('c', data).candidates.includes('connect'));
  assert.deepStrictEqual(T.complete('zzz', data).candidates, []);
});

test('the script has no dynamic code, no HTML strings, one network call and no other storage', () => {
  for (const bad of ['eval(', 'new Function', 'innerHTML', 'outerHTML', 'insertAdjacentHTML', 'document.write', 'XMLHttpRequest',
    'WebSocket', 'sendBeacon', 'document.cookie', 'sessionStorage', 'indexedDB', 'import(', 'importScripts', "setAttribute('style'"]) {
    assert.ok(!source.includes(bad), bad);
  }
  assert.strictEqual(source.split('fetch(').length - 1, 1);
  assert.ok(source.includes("fetch('data.json'"));
});
