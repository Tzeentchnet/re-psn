// Shared framing for the README screenshot. Operates on the real page controls.
// The adapter exposes evaluate(expression) and viewport(width, height).
async function prepareStatusScreenshot(page) {
  await page.viewport(1000, 420); // Preserve the old screenshot's width; grow to show the complete UI.
  await page.evaluate(`document.querySelector('button[data-theme="dark"]').click();
    document.querySelector('#overview-tab').click();document.activeElement.blur()`);
  await page.evaluate('document.fonts.ready.then(() => true)');
  const height = await page.evaluate('Math.ceil(document.documentElement.scrollHeight)');
  await page.viewport(1000, height);
  const fits = await page.evaluate(`['rooms-panel','boards-panel'].every(id => {
    const element=document.getElementById(id), rect=element.getBoundingClientRect();
    return !element.hidden && rect.top>=0 && rect.bottom<=innerHeight;
  }) && document.documentElement.scrollWidth<=innerWidth`);
  if (!fits) throw new Error('Both status tables must fit the README screenshot');
  return {width:1000, height};
}
module.exports = {prepareStatusScreenshot};
