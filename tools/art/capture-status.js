// Takes one screenshot of a web page with headless Chromium through puppeteer. Used by capture-status.sh.
//   node capture-status.js PUPPETEER_DIR URL OUT.png
// PUPPETEER_DIR is the directory of an installed puppeteer package (the one mermaid-cli ships works).
const [dir, url, out] = process.argv.slice(2);
const puppeteer = require(dir);
(async () => {
  const browser = await puppeteer.launch({
    executablePath: process.env.PUPPETEER_EXECUTABLE_PATH || undefined,
    args: ['--no-sandbox'],
  });
  try {
    const page = await browser.newPage();
    await page.setViewport({ width: 1200, height: 800, deviceScaleFactor: 1 });
    await page.emulateMediaFeatures([{ name: 'prefers-color-scheme', value: 'dark' }]);
    await page.goto(url, { waitUntil: 'load' });
    await page.waitForFunction(() => !document.querySelector('#toolbar').hidden);
    await page.screenshot({ path: out });
  } finally {
    await browser.close();
  }
})().catch((e) => { console.error(e); process.exit(1); });
