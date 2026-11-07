const {test} = require('node:test');
const assert = require('node:assert/strict');
const {contrast, colourError, settings, parseStats} = require('../server/status.js');
const colours = {page:'#000000', surface:'#111111', text:'#ffffff', muted:'#dddddd', accent:'#eeeeee', good:'#eeeeee', warn:'#eeeeee', bad:'#eeeeee'};
test('status polling preserves signed scores and unsigned identifiers beyond the safe integer range', () => {
  for (const number of ['9007199254740993', '9223372036854775807', '-9223372036854775808', '18446744073709551615']) {
    assert.equal(parseStats('{"best":'+number+'}').best, number);
  }
  assert.equal(parseStats('{"best":100}').best, 100);
  const text = 'alice "score":9223372036854775807 \\ room';
  assert.equal(parseStats(JSON.stringify({title:text})).title, text);
  assert.throws(() => parseStats('{"best":01}'), SyntaxError);
});
test('contrast follows WCAG luminance and is symmetric', () => {
  assert.equal(contrast('#000000', '#ffffff'), 21);
  assert.equal(contrast('#282828', '#d4be98'), contrast('#d4be98', '#282828'));
});
test('custom themes require all slots, six hex digits and contrast on both backgrounds', () => {
  assert.equal(colourError(colours), '');
  assert.match(colourError({...colours, accent:'url(https://invalid/)'}), /rrggbb/);
  assert.match(colourError({...colours, bad:'#111111'}), /4.5:1/);
  assert.match(colourError({...colours, surface:'#ffffff'}), /surface/);
  assert.match(colourError({...colours, muted:undefined}), /rrggbb/);
});
test('stored settings are bounded and unrelated fields never propagate', () => {
  assert.deepEqual(settings(null), settings('invalid'));
  assert.deepEqual(settings({theme:'unknown', size:'999', density:'broken', token:'secret'}), settings(null));
  const value = settings({theme:'light', size:'18', density:'compact', colours:{light:{...colours, extra:'secret'}, dark:{...colours, text:'#000000'}}});
  assert.equal(value.theme, 'light');
  assert.equal(value.size, '18');
  assert.equal(value.density, 'compact');
  assert.deepEqual(value.colours, {light:colours});
});
