// Records a real gameplay session of Scribblebox (live stack, real API calls).
// Every scene prints a progress line, and the video is always finalised in a
// finally block, so a failure still leaves a playable file.
const { chromium } = require('playwright');

const BASE = process.env.APP_URL || 'http://localhost:3003';
const OUT = process.env.VIDEO_DIR || 'C:/新增資料夾/scribblebox-video/raw';

const say = (message) => { console.log(`[${new Date().toISOString().slice(11, 19)}] ${message}`); };
const wait = (page, ms) => page.waitForTimeout(ms);

async function glideTo(page, selector, ms) {
  const target = page.locator(selector).first();
  if (await target.count() === 0) return false;
  const box = await target.boundingBox();
  if (!box) return false;
  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2, { steps: 26 });
  await wait(page, ms);
  return true;
}

async function notebook(page, phrase, settle = 1800) {
  await page.locator('#notebookInput').click();
  await page.fill('#notebookInput', '');
  await page.type('#notebookInput', phrase, { delay: 80 });
  await wait(page, 450);
  await page.locator('#spawnButton').click();
  await wait(page, settle);
}

async function advance(page, times, pause = 900) {
  for (let i = 0; i < times; i += 1) {
    await page.locator('#btnStep').click();
    await wait(page, pause);
  }
}

async function walk(page, key, times) {
  for (let i = 0; i < times; i += 1) {
    await page.keyboard.press(key);
    await wait(page, 380);
    if (await page.locator('#solvedBanner').isVisible().catch(() => false)) return true;
  }
  return false;
}

async function pickPuzzle(page, index) {
  await page.locator('#puzzleList button').nth(index).click();
  await page.waitForSelector('#player', { timeout: 20000 });
  await wait(page, 1500);
}

(async () => {
  const browser = await chromium.launch();
  const context = await browser.newContext({
    viewport: { width: 1280, height: 800 },
    recordVideo: { dir: OUT, size: { width: 1280, height: 800 } }
  });
  const page = await context.newPage();
  const errors = [];
  page.on('pageerror', (error) => errors.push(String(error)));

  try {
    say('open the game');
    await page.goto(BASE, { waitUntil: 'networkidle' });
    await page.waitForSelector('#puzzleList button', { timeout: 20000 });
    await wait(page, 2000);
    await glideTo(page, '#puzzleList', 1400);
    await glideTo(page, '#notebookInput', 1600);
    await page.mouse.wheel(0, 500); await wait(page, 1300);
    await page.mouse.wheel(0, -500); await wait(page, 1300);

    say('puzzle 1: the notebook writes the world');
    await pickPuzzle(page, 0);
    await glideTo(page, '#puzzleBrief', 1600);
    await notebook(page, 'ladder');
    await walk(page, 'ArrowRight', 3);
    await notebook(page, 'big flaming torch');
    await glideTo(page, '#entityList li', 1800);
    await notebook(page, 'zorble');
    await glideTo(page, '#spawnNote', 2200);
    await advance(page, 3, 950);
    await glideTo(page, '#eventLog li', 2000);

    say('four choices');
    await page.locator('#btnReset').click();
    await wait(page, 1500);
    await glideTo(page, '#hintButton', 1100);
    await page.locator('#hintButton').click();
    let ready = await page.locator('#hintOptions button').first().waitFor({ timeout: 10000 }).then(() => true).catch(() => false);
    if (!ready) {
      await page.locator('#btnReset').click();
      await wait(page, 1300);
      await page.locator('#hintButton').click();
      ready = await page.locator('#hintOptions button').first().waitFor({ timeout: 10000 }).then(() => true).catch(() => false);
    }
    await glideTo(page, '#hintRationale', 1800);
    const choices = await page.locator('#hintOptions button').count();
    for (let i = 0; i < choices; i += 1) {
      await glideTo(page, `#hintOptions button >> nth=${i}`, 1300);
    }
    let solved = false;
    for (let i = 0; i < choices && !solved; i += 1) {
      const button = page.locator('#hintOptions button').nth(i);
      if (await button.count() === 0) continue;
      const phrase = await button.getAttribute('data-phrase');
      await button.click();
      await wait(page, 2300);
      solved = await page.locator('#solvedBanner').isVisible().catch(() => false);
      if (!solved) {
        await glideTo(page, '#hintFeedback', 2000);
        if (await page.locator(`#hintOptions button[data-phrase="${phrase}"]`).count() === 0) break;
      }
    }
    await glideTo(page, '#solvedBanner', 2600);
    await advance(page, 2, 900);
    await glideTo(page, '#goalProgress', 1800);

    say('river: two ways across');
    await pickPuzzle(page, 1);
    await notebook(page, 'bridge', 1900);
    await walk(page, 'ArrowRight', 10);
    await advance(page, 2, 900);
    await glideTo(page, '#goalProgress', 2000);
    await page.locator('#btnReset').click();
    await wait(page, 1400);
    await notebook(page, 'ice', 2000);
    await advance(page, 3, 950);
    await glideTo(page, '#entityList', 1700);
    await walk(page, 'ArrowRight', 10);
    await glideTo(page, '#eventLog li', 1900);

    say('candles');
    await pickPuzzle(page, 2);
    await notebook(page, 'match', 2000);
    await advance(page, 3, 950);
    await notebook(page, 'campfire', 2000);
    await advance(page, 2, 950);
    await glideTo(page, '#goalProgress', 2000);

    say('chest: the key, then dynamite');
    await pickPuzzle(page, 3);
    await notebook(page, 'key', 2100);
    await glideTo(page, '#goalProgress', 1900);
    await page.locator('#btnReset').click();
    await wait(page, 1400);
    await notebook(page, 'dynamite', 1700);
    await notebook(page, 'match', 2200);
    await advance(page, 3, 950);
    await glideTo(page, '#eventLog li', 2200);

    say('the AI panel: key and URL');
    await glideTo(page, '#aiSettings', 1800);
    await page.locator('#aiBaseUrl').click();
    await page.type('#aiBaseUrl', 'http://host.docker.internal:3000/v1', { delay: 28 });
    await wait(page, 500);
    await page.locator('#aiApiKey').click();
    await page.type('#aiApiKey', 'sk-your-key-here', { delay: 28 });
    await wait(page, 700);
    await page.locator('#aiKeyReveal').click();
    await wait(page, 1500);
    await page.locator('#aiKeyReveal').click();
    await wait(page, 800);
    await page.locator('#aiModel').click();
    await page.type('#aiModel', 'gpt-4o-mini', { delay: 28 });
    await wait(page, 700);
    await page.locator('#aiSaveButton').click();
    await wait(page, 1500);
    await glideTo(page, '#aiStatus', 1800);
    await page.locator('#aiTestButton').click();
    await wait(page, 4500);
    await glideTo(page, '#aiTestResult', 3000);
    await glideTo(page, '#aiJudgePanel', 2200);
    await page.locator('#aiJudgeQuestion').click();
    await page.type('#aiJudgeQuestion', 'would a burning rope cut the branch the star sits on?', { delay: 26 });
    await wait(page, 1100);
    await glideTo(page, '#aiPuzzlePanel', 2200);
    await page.locator('#aiPuzzleTheme').click();
    await page.type('#aiPuzzleTheme', 'a lighthouse in a storm', { delay: 26 });
    await wait(page, 1100);
    await page.locator('#aiForgetButton').click();
    await wait(page, 1800);
    await glideTo(page, '#aiStatus', 2000);

    say('closing sweep');
    await page.mouse.wheel(0, 2400); await wait(page, 1500);
    await page.mouse.wheel(0, 2400); await wait(page, 1700);
    await page.mouse.wheel(0, 2400); await wait(page, 2000);
    await page.mouse.wheel(0, -7200); await wait(page, 1600);

    say('page errors: ' + (errors.length ? JSON.stringify(errors.slice(0, 3)) : 'none'));
  } catch (error) {
    say('FAILED: ' + (error && error.message ? error.message : String(error)));
  } finally {
    const video = page.video();
    await context.close().catch(() => {});
    await browser.close().catch(() => {});
    say('video: ' + (video ? await video.path().catch(() => 'unknown') : 'none'));
  }
})();
