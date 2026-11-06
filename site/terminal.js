/* re-psn project site: the terminal-style interface. Hand-written, no dependencies, no build step.

   Without this file the page is an ordinary document. With it, the page is laid out as menu, content and a command line.
   The commands answer from data.json (the one request, to the same origin); nothing is sent anywhere, no cookies are set,
   and the only thing stored is the theme choice (localStorage, optional). Text is built with textContent and DOM nodes,
   never HTML strings, and a link is made only for a github.com address or a path in this repository.

   The first half is pure (run, complete) and is tested with `node --test tools/terminal.test.js`. The second half touches the page. */
(function (root) {
  'use strict';

  var HINT = 'type --help for options';
  var THEMES = ['dark', 'light', 'system'];
  var HELP = [
    ['--help, help', 'this list'],
    ['features [--status S] [id]', 'features with their status; an id gives the detail and evidence'],
    ['roadmap', 'done, now, next, later, not planned'],
    ['settings [name]', 'flags and RE_PSN_* variables with defaults'],
    ['install', 'archives, container, from source'],
    ['connect', 'the ways to connect'],
    ['version', 'the version'],
    ['changelog', 'the latest changes'],
    ['about', 'what re-psn is'],
    ['credits', 'who wrote what'],
    ['licence', 'the licence'],
    ['docs', 'the documentation'],
    ['links', 'repository and related links'],
    ['theme [dark|light|system]', 'appearance'],
    ['clear', 'clear the screen'],
    ['history', 'commands typed in this tab']
  ];
  var COMMANDS = ['--help', 'help', 'features', 'roadmap', 'settings', 'install', 'connect', 'version', 'changelog',
    'about', 'credits', 'licence', 'license', 'docs', 'links', 'theme', 'clear', 'history'];
  var CANONICAL = { '--help': 'help', license: 'licence' };

  // ------------------------------------------------------------------ pure part

  function hrefFor(url, data) {
    // Only github.com or a path in the repository. Anything else is shown as text, not as a link.
    if (typeof url !== 'string') { return null; }
    if (/^https:\/\/github\.com\//.test(url)) { return url; }
    if (/^[a-z][a-z0-9+.-]*:/i.test(url) || url.slice(0, 2) === '//') { return null; }
    if (url.charAt(0) === '#') { return url; }
    return (data && data.repository ? data.repository : 'https://github.com/undecuple/re-psn') + '/blob/main/' + url.replace(/^\/+/, '');
  }

  // `code` and [text](link) in a string -> spans. A span is a string or {t, c, href}.
  function md(text, data) {
    var out = [];
    var re = /`([^`]+)`|\[([^\]]+)\]\(([^)\s]+)\)/g;
    var last = 0;
    var m;
    text = String(text);
    while ((m = re.exec(text)) !== null) {
      if (m.index > last) { out.push(text.slice(last, m.index)); }
      if (m[1] !== undefined) {
        out.push({ t: m[1], c: 'code' });
      } else {
        var h = hrefFor(m[3], data);
        out.push(h ? { t: m[2], href: h } : m[2]);
      }
      last = re.lastIndex;
    }
    if (last < text.length) { out.push(text.slice(last)); }
    return out;
  }

  function link(text, url, data) {
    var h = hrefFor(url, data);
    return h ? { t: text, href: h } : text;
  }

  function err(text) { return [{ t: text, c: 'err' }]; }

  function tagSpan(f) {
    return { t: '[' + f.status_label + ']', c: 'tag ' + (f.status_class || 'dim') };
  }

  function statusList(data) {
    return Object.keys(data.status_counts || {});
  }

  function cmdHelp() {
    var lines = ['Commands', ''];
    HELP.forEach(function (h) { lines.push([{ t: h[0], c: 'cmd' }, '  ' + h[1]]); });
    lines.push('', 'Keys: arrows move, Enter opens, Tab switches pane, / jumps to the command line, Esc goes back, F1 to F4 are shortcuts for help, features, install and settings (F5 still reloads the page).');
    return lines;
  }

  function cmdFeatures(args, data) {
    var status = null;
    var id = null;
    for (var i = 0; i < args.length; i++) {
      var a = args[i];
      if (a === '--status') {
        status = args[i + 1];
        i++;
        if (status === undefined) { return { lines: [err('--status needs a value: ' + statusList(data).join(', '))] }; }
      } else if (a.indexOf('--status=') === 0) {
        status = a.slice(9);
      } else if (a.charAt(0) === '-') {
        return { lines: [err('features: unknown option ' + a.slice(0, 40) + ', use --status ' + statusList(data).join('|'))] };
      } else if (id === null) {
        id = a;
      }
    }
    if (id !== null) {
      var f = data.features.filter(function (x) { return x.id === id; })[0];
      if (!f) { return { lines: [err('features: no feature with id ' + id.slice(0, 40) + ', run features to list them')] }; }
      var lines = [[{ t: f.feature, c: 'strong' }, ' ', { t: f.id, c: 'id' }], [tagSpan(f), ' since ' + (f.since || 'not released')], ''];
      if (f.detail) { lines.push(md(f.detail, data)); }
      lines.push('', ['Evidence: ' + (f.evidence || 'none listed')]);
      if (f.note) { lines.push(['Note: '].concat(md(f.note, data))); }
      lines.push('', [link('All evidence', 'docs/features.md', data)]);
      return { lines: lines };
    }
    if (status !== null && statusList(data).indexOf(status) < 0) {
      return { lines: [err('features: unknown status ' + String(status).slice(0, 40) + ', use ' + statusList(data).join(', '))] };
    }
    var rows = data.features.filter(function (x) { return status === null || x.status === status; });
    var out = [status === null ? rows.length + ' features. features <id> gives the detail.' : rows.length + ' features with status ' + status + '.'];
    var area = null;
    rows.forEach(function (r) {
      if (r.area !== area) { area = r.area; out.push('', [{ t: '# ' + area, c: 'head' }]); }
      out.push([tagSpan(r), ' ' + r.feature + ' ', { t: r.id, c: 'id' }]);
    });
    if (!rows.length) { out.push('Nothing with that status.'); }
    return { lines: out };
  }

  function cmdRoadmap(data) {
    var byId = {};
    data.features.forEach(function (f) { byId[f.id] = f; });
    var out = ['No dates are promised.'];
    Object.keys(data.roadmap).forEach(function (name) {
      var ids = data.roadmap[name];
      if (!ids.length) { return; }
      out.push('', [{ t: '# ' + name + ' (' + ids.length + ')', c: 'head' }]);
      ids.forEach(function (i) {
        var f = byId[i];
        out.push(['  ' + f.feature + ' ', { t: f.id, c: 'id' }].concat(f.status === 'partial' || f.status === 'unverified' ? [' ', tagSpan(f)] : []));
      });
    });
    out.push('', [link('ROADMAP.md has the order and the scope', 'ROADMAP.md', data)]);
    return out;
  }

  function settingLine(s) {
    return [{ t: s.flag, c: 'flag' }, '  ', { t: s.variable, c: 'flag' }, '  default ', { t: s.default, c: 'code' }];
  }

  function cmdSettings(args, data) {
    if (args.length) {
      var q = args[0].toLowerCase().replace(/^-+/, '');
      var hit = data.settings.filter(function (s) {
        return s.flag.toLowerCase().replace(/^-+/, '') === q || s.variable.toLowerCase() === q.toLowerCase();
      })[0];
      if (!hit) { return { lines: [err('settings: no setting named ' + args[0].slice(0, 40) + ', run settings to list them')] }; }
      return { lines: [settingLine(hit), ''].concat([md(hit.description, data)]) };
    }
    var out = [data.settings.length + ' flags; each is also a RE_PSN_<FLAG> variable. A flag wins over the variable. settings <name> shows one.'];
    data.settings.forEach(function (s) {
      out.push('', settingLine(s), '  ');
      out[out.length - 1] = ['  '].concat(md(s.description, data));
    });
    out.push('', [link('docs/running.md has the full text', 'docs/running.md', data)]);
    return { lines: out };
  }

  function cmdInstall(data) {
    var i = data.install;
    var out = [['Latest release: ', link(i.release_tag, i.release_url, data), ' (' + data.stage + ')'], '', [{ t: '# Release archives', c: 'head' }], md(i.archives, data)];
    i.verify_unix.forEach(function (l) { out.push([{ t: l, c: 'code' }]); });
    out.push(md(i.verify_windows, data), '', [{ t: '# Container', c: 'head' }]);
    i.container.forEach(function (l) { out.push([{ t: l, c: 'code' }]); });
    out.push(md(i.container_note, data), '', [{ t: '# From source', c: 'head' }], i.source_note);
    i.source.forEach(function (l) { out.push([{ t: l, c: 'code' }]); });
    return out;
  }

  function cmdConnect(data) {
    var c = data.connect;
    var out = [c.intro, ''];
    c.points.forEach(function (p) { out.push(['- '].concat(md(p, data))); });
    out.push('', md(c.client, data));
    return out;
  }

  function cmdChangelog(data) {
    var ch = data.changelog;
    var out = [];
    var u = ch.unreleased;
    if (u) {
      out.push([{ t: '# Unreleased', c: 'head' }]);
      Object.keys(u.sections).forEach(function (sec) {
        out.push('', sec);
        u.sections[sec].forEach(function (b) { out.push(['- '].concat(md(b, data))); });
      });
      out.push('');
    }
    var l = ch.latest;
    var counts = Object.keys(l.counts).map(function (k) { return l.counts[k] + ' under ' + k; }).join(', ');
    out.push([{ t: '# ' + l.version + (l.date ? ' (' + l.date + ')' : ''), c: 'head' }], counts ? counts + '.' : '');
    out.push('', [link('CHANGELOG.md has the full history', 'CHANGELOG.md', data)]);
    return out;
  }

  function cmdAbout(data) {
    var n = data.status_counts;
    var parts = Object.keys(n).filter(function (k) { return n[k]; }).map(function (k) { return n[k] + ' ' + k; });
    var out = [];
    data.about.paragraphs.forEach(function (p) { out.push(md(p, data), ''); });
    out.push('Version ' + data.version + ' (' + data.stage + '). ' + data.features.length + ' features: ' + parts.join(', ') + '.', '', data.about.notice);
    return out;
  }

  function cmdList(items, data) {
    return items.map(function (p) { return ['- '].concat(md(p, data)); });
  }

  function run(input, data, ctx) {
    ctx = ctx || {};
    var tokens = String(input).trim().split(/\s+/).filter(Boolean);
    if (!tokens.length) { return { lines: [], cmd: '' }; }
    var word = CANONICAL[tokens[0]] || tokens[0];
    var args = tokens.slice(1);
    var res;
    switch (word) {
      case 'help': res = { lines: cmdHelp() }; break;
      case 'features': res = cmdFeatures(args, data); break;
      case 'roadmap': res = { lines: cmdRoadmap(data) }; break;
      case 'settings': res = cmdSettings(args, data); break;
      case 'install': res = { lines: cmdInstall(data) }; break;
      case 'connect': res = { lines: cmdConnect(data) }; break;
      case 'version':
        res = { lines: ['re-psn v' + data.version + ' (' + data.stage + ')', data.go ? 'Go ' + data.go + ' or newer to build.' : ''].filter(Boolean) };
        break;
      case 'changelog': res = { lines: cmdChangelog(data) }; break;
      case 'about': res = { lines: cmdAbout(data) }; break;
      case 'credits': res = { lines: cmdList(data.credits, data) }; break;
      case 'licence': res = { lines: [md(data.licence, data)] }; break;
      case 'docs':
        res = { lines: data.docs.map(function (d) { return [link(d.title, d.url, data), '  ' + d.summary]; }) };
        break;
      case 'links':
        res = { lines: data.links.map(function (l) { return [l.title + '  ', link(l.url, l.url, data)]; }) };
        break;
      case 'theme':
        if (!args.length) {
          res = { lines: ['theme: ' + (ctx.theme || 'system') + ' (dark, light or system)'] };
        } else if (THEMES.indexOf(args[0]) < 0) {
          res = { lines: [err('theme: unknown value ' + args[0].slice(0, 40) + ', use dark, light or system')] };
        } else {
          res = { lines: ['theme: ' + args[0]], theme: args[0] };
        }
        break;
      case 'clear': res = { lines: [], clear: true }; break;
      case 'history':
        res = { lines: (ctx.history && ctx.history.length) ? ctx.history.map(function (h, i) { return (i + 1) + '  ' + h; }) : ['No history yet.'] };
        break;
      default:
        return { lines: [err('command not found: ' + tokens[0].slice(0, 60) + ', ' + HINT)], cmd: '', error: true };
    }
    res.cmd = word;
    return res;
  }

  function commonPrefix(list) {
    var p = list[0];
    list.forEach(function (s) { while (s.indexOf(p) !== 0) { p = p.slice(0, -1); } });
    return p;
  }

  // Tab completion: {value, candidates}. value is the new text of the input.
  function complete(input, data) {
    var endsWithSpace = /\s$/.test(input);
    var tokens = input.split(/\s+/).filter(Boolean);
    var cur = endsWithSpace ? '' : (tokens.pop() || '');
    var head = tokens.join(' ');
    var pool;
    if (!tokens.length) {
      pool = COMMANDS;
    } else {
      var word = CANONICAL[tokens[0]] || tokens[0];
      var prev = tokens[tokens.length - 1];
      if (word === 'features') {
        pool = prev === '--status' ? statusList(data) : ['--status'].concat(data.features.map(function (f) { return f.id; }));
      } else if (word === 'settings') {
        pool = data.settings.map(function (s) { return s.flag; });
      } else if (word === 'theme') {
        pool = THEMES;
      } else {
        pool = [];
      }
    }
    var c = pool.filter(function (x) { return x.indexOf(cur) === 0; });
    if (!c.length) { return { value: input, candidates: [] }; }
    var pre = c.length === 1 ? c[0] + ' ' : commonPrefix(c);
    return { value: (head ? head + ' ' : '') + pre, candidates: c.length > 1 ? c : [] };
  }

  var api = { run: run, complete: complete, md: md, hrefFor: hrefFor, COMMANDS: COMMANDS, HINT: HINT };
  if (typeof module !== 'undefined' && module.exports) { module.exports = api; }
  root.rePsnTerminal = api;

  // ------------------------------------------------------------------ the page

  if (typeof document === 'undefined' || typeof fetch !== 'function') { return; }

  var MENU = [['about', 'Overview'], ['features', 'Features'], ['roadmap', 'Roadmap'], ['install', 'Install'],
    ['settings', 'Settings'], ['connect', 'Connect'], ['changelog', 'Changelog'], ['docs', 'Docs']];
  var FKEYS = { F1: 'help', F2: 'features', F3: 'install', F4: 'settings', F5: 'connect' };
  var STORE = 're-psn-theme';

  function el(tag, cls, text) {
    var n = document.createElement(tag);
    if (cls) { n.className = cls; }
    if (text !== undefined) { n.textContent = text; }
    return n;
  }

  function storedTheme() {
    try {
      var t = root.localStorage.getItem(STORE);
      return THEMES.indexOf(t) >= 0 ? t : 'system';
    } catch (e) { return 'system'; }
  }

  // The one theme state: data-theme on the root, the stored choice, and every System / Light / Dark control on the page.
  var theme = 'system';

  function syncControls() {
    var btns = document.querySelectorAll('[data-theme-set]');
    for (var i = 0; i < btns.length; i++) {
      btns[i].setAttribute('aria-pressed', btns[i].getAttribute('data-theme-set') === theme ? 'true' : 'false');
    }
  }

  function applyTheme(t, save) {
    var de = document.documentElement;
    theme = THEMES.indexOf(t) >= 0 ? t : 'system';
    if (theme === 'dark' || theme === 'light') { de.setAttribute('data-theme', theme); } else { de.removeAttribute('data-theme'); }
    if (save) {
      try {
        if (theme === 'system') { root.localStorage.removeItem(STORE); } else { root.localStorage.setItem(STORE, theme); }
      } catch (e) { /* storage is optional */ }
    }
    syncControls();
  }

  // A System / Light / Dark button group. The one in the page header is generated by tools/site.py; the interface makes its own.
  function themeControl() {
    var g = el('div', 'theme');
    g.setAttribute('role', 'group');
    g.setAttribute('aria-label', 'Theme');
    g.appendChild(el('span', 'theme-label', 'Theme'));
    [['system', 'System'], ['light', 'Light'], ['dark', 'Dark']].forEach(function (o) {
      var b = el('button', 'theme-btn', o[1]);
      b.type = 'button';
      b.setAttribute('data-theme-set', o[0]);
      b.setAttribute('aria-pressed', 'false');
      g.appendChild(b);
    });
    return g;
  }

  // Runs as soon as the script does (it is deferred, so after the page is parsed and before data.json arrives).
  applyTheme(storedTheme(), false);
  document.documentElement.classList.add('js');
  document.addEventListener('click', function (e) {
    var b = e.target.closest ? e.target.closest('[data-theme-set]') : null;
    if (b) { applyTheme(b.getAttribute('data-theme-set'), true); }
  });
  syncControls();

  function renderLine(line) {
    var d = el('div', 'ln');
    (Array.isArray(line) ? line : [line]).forEach(function (sp) {
      if (typeof sp === 'string') { d.appendChild(document.createTextNode(sp)); return; }
      var h = sp.href ? hrefFor(sp.href, null) : null;
      var n;
      if (h) {
        n = el('a', sp.c, sp.t);
        n.setAttribute('href', h);
        n.setAttribute('rel', 'noopener noreferrer');
      } else {
        n = el('span', sp.c, sp.t);
      }
      d.appendChild(n);
    });
    return d;
  }

  function mount(data) {
    var history = [];
    var histPos = 0;
    var current = '';

    var tui = el('div', 'tui');
    tui.id = 'tui';
    tui.setAttribute('data-view', 'content');

    var title = el('div', 'tui-title');
    title.appendChild(el('span', 'tui-name', 're-psn'));
    title.appendChild(el('span', 'tui-ver', 'v' + data.version + ' · ' + data.stage));
    var plain = el('button', 'tui-plain', 'plain view');
    plain.type = 'button';
    title.appendChild(themeControl());
    title.appendChild(plain);
    tui.appendChild(title);

    var fk = el('ul', 'tui-fkeys');
    fk.setAttribute('aria-label', 'Shortcuts');
    Object.keys(FKEYS).forEach(function (k) {
      var li = el('li');
      var b = el('button', 'fk', k === 'F5' ? '[' + FKEYS[k] + ']' : '[' + k + ' ' + FKEYS[k] + ']');
      b.type = 'button';
      b.setAttribute('data-cmd', FKEYS[k]);
      li.appendChild(b);
      fk.appendChild(li);
    });
    tui.appendChild(fk);

    var panes = el('div', 'tui-panes');
    var nav = el('nav', 'tui-menu');
    nav.setAttribute('aria-label', 'Sections');
    var ul = el('ul');
    var menuBtns = MENU.map(function (m) {
      var li = el('li');
      var b = el('button', 'mi', m[1]);
      b.type = 'button';
      b.setAttribute('data-cmd', m[0]);
      li.appendChild(b);
      ul.appendChild(li);
      return b;
    });
    nav.appendChild(ul);
    panes.appendChild(nav);

    var content = el('div', 'tui-content');
    var back = el('button', 'chip tui-back', '← Back to menu');
    back.type = 'button';
    content.appendChild(back);
    var out = el('div', 'tui-out');
    out.setAttribute('role', 'region');
    out.setAttribute('aria-label', 'Output');
    out.setAttribute('aria-live', 'polite');
    out.tabIndex = 0;
    content.appendChild(out);
    panes.appendChild(content);
    tui.appendChild(panes);

    var form = el('form', 'tui-cmd');
    form.setAttribute('autocomplete', 'off');
    form.setAttribute('action', '#');
    var hint = el('p', 'tui-hint', HINT);
    hint.id = 'tui-hint';
    form.appendChild(hint);
    var row = el('div', 'tui-row');
    var ps = el('span', 'tui-ps', '$');
    ps.setAttribute('aria-hidden', 'true');
    var label = el('label', 'vh', 'Command');
    label.setAttribute('for', 'tui-input');
    var input = el('input');
    input.id = 'tui-input';
    input.type = 'text';
    input.maxLength = 200;
    input.spellcheck = false;
    input.setAttribute('autocapitalize', 'off');
    input.setAttribute('autocorrect', 'off');
    input.setAttribute('autocomplete', 'off');
    input.setAttribute('aria-describedby', 'tui-hint');
    var cursor = el('span', 'cursor', '█');
    cursor.setAttribute('aria-hidden', 'true');
    row.appendChild(ps);
    row.appendChild(label);
    row.appendChild(input);
    row.appendChild(cursor);
    form.appendChild(row);
    tui.appendChild(form);

    var chips = el('ul', 'tui-chips');
    chips.setAttribute('aria-label', 'Commands');
    ['help', 'features', 'roadmap', 'settings', 'install', 'connect', 'changelog', 'docs', 'links', 'theme'].forEach(function (c) {
      var li = el('li');
      var b = el('button', 'chip', c);
      b.type = 'button';
      b.setAttribute('data-cmd', c);
      li.appendChild(b);
      chips.appendChild(li);
    });
    tui.appendChild(chips);

    tui.appendChild(el('p', 'tui-keys', '↑↓ move · Enter open · Tab switch pane · / command · Esc back'));

    var toTui = el('div', 'plain-bar');
    var tb = el('button', 'tui-plain', 'terminal view');
    tb.type = 'button';
    toTui.appendChild(tb);

    function setView(v) { tui.setAttribute('data-view', v); }

    function paint(res, echo) {
      if (res.clear) { out.textContent = ''; return; }
      out.textContent = '';
      if (echo) { out.appendChild(renderLine([{ t: '$ ', c: 'ps' }, { t: echo, c: 'echo' }])); }
      res.lines.forEach(function (l) { out.appendChild(renderLine(l)); });
      out.scrollTop = 0;
    }

    function mark(cmd) {
      menuBtns.forEach(function (b) {
        if (b.getAttribute('data-cmd') === cmd) { b.setAttribute('aria-current', 'page'); } else { b.removeAttribute('aria-current'); }
      });
    }

    function exec(text, focusOut) {
      var t = String(text).trim();
      if (!t) { return; }
      if (history[history.length - 1] !== t) { history.push(t); }
      histPos = history.length;
      var res = run(t, data, { history: history, theme: theme });
      if (res.theme) { applyTheme(res.theme, true); }
      paint(res, t);
      if (res.cmd) { current = res.cmd; mark(current); }
      setView('content');
      if (focusOut) { out.focus(); }
    }

    function menuFocus() {
      var b = menuBtns.filter(function (x) { return x.getAttribute('aria-current') === 'page'; })[0] || menuBtns[0];
      setView('menu');
      b.focus();
    }

    function syncCursor() {
      var empty = input.value === '';
      tui.classList.toggle('empty', empty);
    }

    tui.addEventListener('click', function (e) {
      var b = e.target.closest ? e.target.closest('button[data-cmd]') : null;
      if (b && tui.contains(b)) { exec(b.getAttribute('data-cmd'), true); }
    });
    back.addEventListener('click', menuFocus);
    plain.addEventListener('click', function () {
      document.documentElement.classList.add('plain');
      tb.focus();
    });
    tb.addEventListener('click', function () {
      document.documentElement.classList.remove('plain');
      input.focus();
    });

    form.addEventListener('submit', function (e) {
      e.preventDefault();
      var v = input.value;
      input.value = '';
      syncCursor();
      exec(v, false);
    });

    input.addEventListener('input', syncCursor);
    input.addEventListener('keydown', function (e) {
      if (e.altKey || e.ctrlKey || e.metaKey) { return; }
      if (e.key === 'ArrowUp' || e.key === 'ArrowDown') {
        if (!history.length) { return; }
        e.preventDefault();
        histPos += e.key === 'ArrowUp' ? -1 : 1;
        histPos = Math.max(0, Math.min(history.length, histPos));
        input.value = history[histPos] || '';
        syncCursor();
      } else if (e.key === 'Tab' && !e.shiftKey && input.value.trim() !== '') {
        var c = complete(input.value, data);
        if (c.value !== input.value || c.candidates.length) {
          e.preventDefault();
          input.value = c.value;
          syncCursor();
          if (c.candidates.length) { paint({ lines: [c.candidates.join('  ')] }, null); }
        }
      } else if (e.key === 'Escape') {
        e.preventDefault();
        menuFocus();
      }
    });

    nav.addEventListener('keydown', function (e) {
      var i = menuBtns.indexOf(document.activeElement);
      if (i < 0 || e.altKey || e.ctrlKey || e.metaKey) { return; }
      var n = null;
      if (e.key === 'ArrowDown') { n = (i + 1) % menuBtns.length; }
      if (e.key === 'ArrowUp') { n = (i + menuBtns.length - 1) % menuBtns.length; }
      if (e.key === 'Home') { n = 0; }
      if (e.key === 'End') { n = menuBtns.length - 1; }
      if (n !== null) { e.preventDefault(); menuBtns[n].focus(); }
    });

    document.addEventListener('keydown', function (e) {
      if (e.altKey || e.ctrlKey || e.metaKey || e.shiftKey) { return; }
      if (document.documentElement.classList.contains('plain')) { return; }
      if (FKEYS[e.key] && e.key !== 'F5') { e.preventDefault(); exec(FKEYS[e.key], false); return; }
      var t = e.target;
      var typing = t && (t.tagName === 'INPUT' || t.tagName === 'TEXTAREA' || t.isContentEditable);
      if (e.key === '/' && !typing) { e.preventDefault(); input.focus(); }
      if (e.key === 'Escape' && !typing) { menuFocus(); }
    });

    var anchor = document.querySelector('.skip');
    document.body.insertBefore(toTui, anchor ? anchor.nextSibling : document.body.firstChild);
    toTui.parentNode.insertBefore(tui, toTui.nextSibling);
    document.documentElement.classList.add('tui-on');
    syncCursor();
    syncControls();
    paint(run('about', data, {}), null);
    current = 'about';
    mark(current);
    if (root.matchMedia && root.matchMedia('(max-width: 44rem)').matches) { setView('menu'); }
  }

  function start() {
    fetch('data.json', { credentials: 'same-origin' })
      .then(function (r) { if (!r.ok) { throw new Error('data.json'); } return r.json(); })
      .then(mount)
      .catch(function () { /* the plain page stays as it is */ });
  }

  if (document.readyState === 'loading') { document.addEventListener('DOMContentLoaded', start); } else { start(); }
})(typeof window !== 'undefined' ? window : globalThis);
