// Records the short AI segment: a real key, a real endpoint, the judge, a generated
// puzzle and the AI four choices. The key is read from the environment, is never
// printed, and is never revealed in the UI (the reveal button is left alone), so it
// cannot end up on screen or in this file.
const { chromium } = require('playwright');

const BASE = process.env.APP_URL || 'http://localhost:3003';
const OUT = process.env.VIDEO_DIR || 'C:/新增資料夾/scribblebox-video/raw-ai';
const KEY = process.env.AI_KEY || '';
const AI_BASE = process.env.AI_BASE || 'http://host.docker.internal:3000/v1';
const AI_MODEL = process.env.AI_MODEL || 'gpt-4o-mini';

const say = (message) => { console.log(`[${new Date().toISOString().slice(11, 19)}] ${message}`); };
const wait = (page, ms) => page.waitForTimeout(ms);

if (!KEY) {
  console.error('AI_KEY is not set: refusing to record an AI segment without a key.');
  process.exit(2);
}

async function glideTo(page, selector, ms) {
  const target = page.locator(selector).first();
  if (await target.count() === 0) return false;
  const box = await target.boundingBox();
  if (!box) return false;
  await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2, { steps: 24 });
  await wait(page, ms);
  return true;
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

  let leaked = false;
  const watchForLeak = async (label) => {
    const html = await page.content();
    if (html.includes(KEY)) {
      leaked = true;
      console.error(`LEAK: the key is visible in the page after ${label}`);
    }
  };

  try {
    say('open the game and pick the star puzzle');
    await page.goto(BASE, { waitUntil: 'networkidle' });
    await page.waitForSelector('#puzzleList button', { timeout: 20000 });
    await page.locator('#puzzleList button').first().click();
    await page.waitForSelector('#player', { timeout: 20000 });
    await wait(page, 1500);

    say('fill the AI settings (key stays masked)');
    await glideTo(page, '#aiSettings', 1600);
    await page.locator('#aiToggle').check();
    await wait(page, 500);
    await page.locator('#aiBaseUrl').click();
    await page.type('#aiBaseUrl', AI_BASE, { delay: 26 });
    await wait(page, 500);
    await page.locator('#aiApiKey').click();
    await page.type('#aiApiKey', KEY, { delay: 22 });
    await wait(page, 700);
    await page.locator('#aiModel').click();
    await page.type('#aiModel', AI_MODEL, { delay: 26 });
    await wait(page, 700);
    await glideTo(page, '#aiModel', 800);
    await page.locator('#aiSaveButton').click();
    await wait(page, 1500);
    await glideTo(page, '#aiStatus', 1800);
    await watchForLeak('saving the settings');

    say('test the connection for real');
    await glideTo(page, '#aiTestButton', 800);
    await page.locator('#aiTestButton').click();
    await page.waitForFunction(
      () => {
        const box = document.querySelector('#aiTestResult');
        return box && /Working:|did not work/.test(box.textContent || '');
      },
      null,
      { timeout: 90000 }
    );
    await glideTo(page, '#aiTestResult', 2800);
    await watchForLeak('testing the connection');

    // The four choices first: an approved judgement solves the world, and a solved
    // puzzle has nothing left to hint at.
    say('the AI four choices');
    await glideTo(page, '#hintPanel', 900);
    await page.locator('#hintButton').click();
    await page.locator('#hintOptions button').first().waitFor({ timeout: 20000 }).catch(() => {});
    await wait(page, 1200);
    const aiHint = page.locator('#aiHintButton');
    if (await aiHint.count() && await aiHint.isEnabled()) {
      await glideTo(page, '#aiHintButton', 800);
      await aiHint.click();
      await page.waitForTimeout(6000);
      await glideTo(page, '#hintOptions', 3200);
      await glideTo(page, '#hintRationale', 2400);
    }
    await watchForLeak('the AI four choices');

    say('ask the AI judge');
    await page.locator('#aiJudgeQuestion').click();
    await page.type('#aiJudgeQuestion', 'I stack three boxes under the tree and climb up — does that count?', { delay: 24 });
    await wait(page, 900);
    await glideTo(page, '#aiJudgeButton', 700);
    await page.locator('#aiJudgeButton').click();
    await page.waitForFunction(
      () => {
        const box = document.querySelector('#aiVerdict');
        return box && /approved|not|denied|No answer|The model|judge/i.test(box.textContent || '');
      },
      null,
      { timeout: 120000 }
    ).catch(() => {});
    await wait(page, 800);
    await glideTo(page, '#aiVerdict', 4200);
    await glideTo(page, '#solvedBanner', 2200);
    await watchForLeak('asking the judge');

    say('invent a puzzle with AI');
    await glideTo(page, '#aiPuzzlePanel', 1800);
    await page.locator('#aiPuzzleTheme').click();
    await page.type('#aiPuzzleTheme', 'a lighthouse in a storm', { delay: 24 });
    await wait(page, 700);
    await page.locator('#aiPuzzleGoalKind').selectOption('star').catch(() => {});
    await wait(page, 700);
    await glideTo(page, '#aiPuzzleButton', 800);
    // A slower model sometimes needs a second try; ask again rather than showing a
    // failure in the recording, exactly as a player would.
    let invented = false;
    for (let attempt = 1; attempt <= 2 && !invented; attempt += 1) {
      await page.locator('#aiPuzzleButton').click();
      await page.waitForFunction(
        () => {
          const status = document.querySelector('#aiPuzzleStatus');
          const error = document.querySelector('#aiPuzzleError');
          const badge = document.querySelector('#aiWorldBadge');
          const text = (status ? status.textContent : '') + (error ? error.textContent : '');
          return (badge && !badge.hidden && badge.textContent.trim() !== '') || /rejected|did not|could not|no four|refused/i.test(text || '');
        },
        null,
        { timeout: 120000 }
      ).catch(() => {});
      await wait(page, 1500);
      invented = await page.locator('#aiWorldBadge').isVisible().catch(() => false);
      if (!invented) say('invent attempt ' + attempt + ' did not land; asking again');
    }
    await glideTo(page, '#worldHeading', 2600);
    await glideTo(page, '#aiWorldBadge', 2400);
    await glideTo(page, '#goalBanner', 2000);
    await watchForLeak('inventing a puzzle');

    say('play the invented puzzle a little');
    const notebook = async (phrase) => {
      await page.locator('#notebookInput').click();
      await page.fill('#notebookInput', '');
      await page.type('#notebookInput', phrase, { delay: 70 });
      await wait(page, 400);
      await page.locator('#spawnButton').click();
      await wait(page, 1800);
    };
    await notebook('ladder');
    await notebook('rope');
    for (let i = 0; i < 6; i += 1) {
      await page.keyboard.press('ArrowRight');
      await wait(page, 350);
      if (await page.locator('#solvedBanner').isVisible().catch(() => false)) break;
    }
    for (let i = 0; i < 6; i += 1) {
      await page.keyboard.press('Space');
      await wait(page, 350);
      if (await page.locator('#solvedBanner').isVisible().catch(() => false)) break;
    }
    await page.locator('#btnStep').click();
    await wait(page, 1500);
    await glideTo(page, '#goalProgress', 2400);

    say('page errors: ' + (errors.length ? JSON.stringify(errors.slice(0, 3)) : 'none'));
    say('key leaked to screen: ' + leaked);
  } catch (error) {
    say('FAILED: ' + (error && error.message ? error.message : String(error)));
  } finally {
    const video = page.video();
    await context.close().catch(() => {});
    await browser.close().catch(() => {});
    say('video: ' + (video ? await video.path().catch(() => 'unknown') : 'none'));
  }
})();
