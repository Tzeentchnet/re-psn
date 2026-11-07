// Optional browser-test support using the Chromium DevTools Protocol; Node built-ins only.
import {writeFile} from 'node:fs/promises';

export async function connect(port) {
  const page = await (await fetch(`http://127.0.0.1:${port}/json/new?about:blank`, {method:'PUT',signal:AbortSignal.timeout(10000)})).json();
  const ws = new WebSocket(page.webSocketDebuggerUrl);
  await new Promise((resolve, reject) => {
    const timer=setTimeout(()=>{ws.close();reject(new Error('Browser connection timed out'));},10000);
    ws.onopen=()=>{clearTimeout(timer);resolve();};
    ws.onerror=error=>{clearTimeout(timer);reject(error);};
  });
  let next = 0;
  const pending = new Map();
  const events = [];
  ws.addEventListener('message', ({data}) => {
    const m = JSON.parse(data);
    if (m.id) {
      const p = pending.get(m.id);
      pending.delete(m.id);
      if (p) clearTimeout(p.timer);
      if (p) m.error ? p.reject(new Error(JSON.stringify(m.error))) : p.resolve(m.result);
    } else events.push(m);
  });
  ws.addEventListener('close',()=>{
    for(const p of pending.values()) {clearTimeout(p.timer);p.reject(new Error('Browser connection closed'));}
    pending.clear();
  });
  const call = (method, params = {}) => new Promise((resolve, reject) => {
    const id = ++next;
    const timer=setTimeout(()=>{pending.delete(id);reject(new Error('Browser command timed out: '+method));},10000);
    pending.set(id, {resolve, reject, timer});
    ws.send(JSON.stringify({id, method, params}));
  });
  const evaluate = async expression => {
    const r = await call('Runtime.evaluate', {expression, returnByValue: true, awaitPromise: true});
    if (r.exceptionDetails) throw new Error(JSON.stringify(r.exceptionDetails));
    return r.result.value;
  };
  const waitFor = async expression => {
    const end = Date.now() + 10000;
    while (Date.now() < end) {
      if (await evaluate(expression)) return;
      await new Promise(r => setTimeout(r, 30));
    }
    throw new Error('Timed out: ' + expression);
  };
  const key = async (key, code = key, extra = {}) => {
    const vk = {Enter:13,Tab:9,Escape:27,PageDown:34,PageUp:33,Home:36,End:35,ArrowUp:38,ArrowDown:40,ArrowLeft:37,ArrowRight:39}[key];
    const props = {key,code,windowsVirtualKeyCode:vk,nativeVirtualKeyCode:vk,...(key==='Enter'?{text:'\r',unmodifiedText:'\r'}:{}),...extra};
    await call('Input.dispatchKeyEvent', {type: 'keyDown',...props});
    await call('Input.dispatchKeyEvent', {type: 'keyUp',...props});
  };
  const command = async text => {
    await evaluate(`document.querySelector('#tui-input').value = ${JSON.stringify(text)};
      document.querySelector('#tui-input').dispatchEvent(new Event('input', {bubbles:true}));
      document.querySelector('.tui-cmd').requestSubmit()`);
  };
  const screenshot = async path => {
    const r = await call('Page.captureScreenshot', {format: 'png', captureBeyondViewport: false});
    await writeFile(path, Buffer.from(r.data, 'base64'));
  };
  await call('Page.enable');
  await call('Runtime.enable');
  await call('Network.enable');
  await call('Network.setCacheDisabled', {cacheDisabled:true});
  return {call, evaluate, waitFor, key, command, screenshot, events, close: async () => { await call('Page.close'); ws.close(); }};
}
