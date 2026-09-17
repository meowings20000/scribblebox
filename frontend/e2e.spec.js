/* Playwright specs for the Scribblebox frontend.
 *
 * `APP_URL` points at whatever is serving frontend/ :
 *   - the whole docker stack (npm run test:e2e, APP_URL=http://localhost:3003) for the live tests
 *   - any static server (python -m http.server 4173 -d frontend) for the mocked tests, which
 *     intercept every /api/** call and therefore need no backend at all.
 *
 * The live tests are written against the HTTP contract in the brief and skip nothing: they fail
 * loudly if the API is not there, which is what a live test should do.
 */
const { test, expect } = require('@playwright/test');

const APP_URL = process.env.APP_URL || 'http://localhost:3003';

const PUZZLE = {
  id: 'star-in-tree',
  title: 'The star in the tree',
  brief: 'Get the star out of the tree.',
  hint: 'Wood burns. Rope pulls. Think about what you can stand on.',
  goalKind: 'star',
  width: 28,
  height: 16
};

/* ------------------------------------------------------------------ fixtures (contract shape) */

function objectOf(overrides = {}) {
  return Object.assign({
    key: 'ladder',
    name: 'ladder',
    noun: 'ladder',
    category: 'structure',
    approximate: false,
    modifiers: [],
    tags: ['climbable', 'solid', 'platform'],
    mass: 2,
    size: 2,
    color: '#c98a3c',
    shape: 'ladder',
    glyph: 'LAD',
    note: ''
  }, overrides);
}

function terrainOf(width, height) {
  const terrain = new Array(width * height).fill(0);
  for (let x = 0; x < width; x += 1) {
    for (let y = 12; y < height; y += 1) terrain[y * width + x] = 1;
  }
  for (let y = 9; y < 12; y += 1) terrain[y * width + 4] = 3;
  for (let y = 11; y < 12; y += 1) terrain[y * width + 4] = 3;
  return terrain;
}

function worldState(overrides = {}) {
  const width = 28;
  const height = 16;
  return Object.assign({
    id: 'a1b2c3d4e5f6',
    puzzle: PUZZLE,
    width: width,
    height: height,
    terrain: terrainOf(width, height),
    entities: [{
      id: 1,
      object: objectOf({ key: 'star', name: 'star', category: 'goal', tags: ['goal', 'grab'], shape: 'star', glyph: 'STAR', color: '#f5c542' }),
      x: 4, y: 8, w: 1, h: 1,
      burning: false, opened: false, cut: false, powered: false, held: false
    }],
    player: { x: 2, y: 11, onGround: true, holding: 0 },
    solved: false,
    ticks: 12,
    events: ['the star is stuck in the tree'],
    notebook: [],
    goal: { title: 'Get the star out of the tree.', met: false, progress: 'the star is still up there' }
  }, overrides);
}

const HINT = {
  placement: { x: 6, y: 3 },
  options: ['ladder', 'cake', 'pillow', 'umbrella'],
  rationale: 'one of these will do the job'
};

const AI_SPEC = {
  id: 'ai-lighthouse-storm',
  title: 'The lighthouse in the storm',
  brief: 'Reach the lamp room before the tide.',
  hint: 'Something that floats, or something that climbs.',
  goalKind: 'star',
  width: 24,
  height: 14,
  terrain: terrainOf(24, 14),
  entities: [{ phrase: 'ladder', x: 3, y: 4 }],
  playerX: 2,
  playerY: 11
};

/* ------------------------------------------------------------------ mock helpers */

async function json(route, status, body) {
  return route.fulfill({ status, contentType: 'application/json', body: JSON.stringify(body) });
}

/* Serves the whole contract from memory, so the frontend can be exercised without a backend.
 * `overrides` keys are "METHOD /path" and may hold a {status, body} pair or a handler. */
async function mockApi(page, overrides = {}) {
  // One mutable world per page: spawning, stepping and acting change it like the API would.
  const session = {
    world: worldState(),
    spawns: [],
    steps: [],
    acts: [],
    hints: 0,
    aiWorlds: 0
  };

  await page.route('**/api/**', async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const key = request.method() + ' ' + url.pathname;
    if (Object.prototype.hasOwnProperty.call(overrides, key)) {
      const override = overrides[key];
      if (typeof override === 'function') return override(route, url, session);
      return json(route, override.status, override.body);
    }

    const path = url.pathname;
    if (path === '/api/health') return json(route, 200, { status: 'ok', words: 180, puzzles: 1 });
    if (path === '/api/puzzles') return json(route, 200, { puzzles: [PUZZLE] });

    if (path === '/api/words') {
      const query = (url.searchParams.get('q') || '').toLowerCase();
      const known = ['ladder', 'star', 'cake', 'pillow', 'umbrella', 'rope'];
      const shapes = { ladder: 'ladder', star: 'star', cake: 'blob', pillow: 'blob', umbrella: 'tool', rope: 'rope' };
      const body = {
        words: known.filter((word) => word.startsWith(query)),
        exact: known.indexOf(query) !== -1,
        dictionary: 180,
        alwaysWorks: true
      };
      if (body.exact) body.match = { key: query, shape: shapes[query] || 'box', color: '#c9a06a', glyph: query.slice(0, 3).toUpperCase(), note: 'a mocked word' };
      return json(route, 200, body);
    }

    if (path === '/api/worlds' && request.method() === 'POST') {
      const body = JSON.parse(request.postData() || '{}');
      if (body.aiSpec) {
        session.aiWorlds += 1;
        session.world = worldState({
          id: 'a1b2c3d4e5f6',
          puzzle: {
            id: body.aiSpec.id,
            title: body.aiSpec.title,
            brief: body.aiSpec.brief,
            hint: body.aiSpec.hint,
            goalKind: body.aiSpec.goalKind,
            width: body.aiSpec.width,
            height: body.aiSpec.height
          },
          width: body.aiSpec.width,
          height: body.aiSpec.height,
          terrain: body.aiSpec.terrain,
          entities: [],
          goal: { title: body.aiSpec.brief, met: false, progress: 'the lamp room is still dark' }
        });
        return json(route, 200, { state: session.world });
      }
      session.world = worldState();
      return json(route, 200, { state: session.world });
    }

    if (/^\/api\/worlds\/[^/]+\/spawn$/.test(path)) {
      const body = JSON.parse(request.postData() || '{}');
      session.spawns.push(body);
      const key = String(body.phrase || '').split(/\s+/).pop();
      const spawned = objectOf({ key: key, name: body.phrase, noun: key, glyph: key.slice(0, 3).toUpperCase() });
      // The engine's own truth: only the ladder reaches the star here.
      const solved = key === 'ladder';
      const entity = {
        id: session.world.entities.length + 2,
        object: spawned,
        x: body.x, y: body.y, w: 1, h: 2,
        burning: false, opened: false, cut: false, powered: false, held: false
      };
      session.world = worldState({
        entities: session.world.entities.concat([entity]),
        ticks: session.world.ticks,
        events: ['a ' + body.phrase + ' appears'],
        notebook: session.world.notebook.concat([{ phrase: body.phrase, key: key, approximate: false, x: body.x, y: body.y }]),
        solved: solved,
        goal: { title: PUZZLE.brief, met: solved, progress: solved ? 'the star is out of the tree' : 'the star is still up there' }
      });
      return json(route, 200, {
        spawned: spawned,
        note: 'a ' + body.phrase + ' appears',
        approximate: false,
        events: ['a ' + body.phrase + ' appears'],
        state: session.world
      });
    }

    if (/^\/api\/worlds\/[^/]+\/step$/.test(path)) {
      const body = JSON.parse(request.postData() || '{}');
      session.steps.push(body);
      session.world = Object.assign({}, session.world, {
        ticks: session.world.ticks + (Number(body.ticks) || 0),
        events: ['time passes']
      });
      return json(route, 200, { events: ['time passes'], state: session.world });
    }

    if (/^\/api\/worlds\/[^/]+\/act$/.test(path)) {
      const body = JSON.parse(request.postData() || '{}');
      session.acts.push(body);
      const player = Object.assign({}, session.world.player);
      if (body.action === 'right') player.x = player.x + 1;
      if (body.action === 'left') player.x = player.x - 1;
      if (body.action === 'jump') player.y = player.y - 1;
      session.world = Object.assign({}, session.world, {
        player: player,
        ticks: session.world.ticks + 1,
        events: ['you ' + body.action]
      });
      return json(route, 200, { events: ['you ' + body.action], state: session.world });
    }

    if (/^\/api\/worlds\/[^/]+\/reset$/.test(path)) {
      session.world = worldState();
      return json(route, 200, { events: ['the world resets'], state: session.world });
    }

    if (/^\/api\/worlds\/[^/]+\/hint$/.test(path)) {
      session.hints += 1;
      return json(route, 200, { hint: HINT, state: session.world });
    }

    // A choice is placed by the server, which re-proves the spot for the world as
    // it stands. The mock mirrors that: the choice lands on the offered spot, and
    // only the ladder wins here.
    if (/^\/api\/worlds\/[^/]+\/hint\/choose$/.test(path)) {
      const body = JSON.parse(request.postData() || '{}');
      session.choices = session.choices || [];
      session.choices.push(body);
      session.spawns.push({ phrase: body.phrase, x: HINT.placement.x, y: HINT.placement.y });
      const key = String(body.phrase || '').split(/\s+/).pop();
      const spawned = objectOf({ key: key, name: body.phrase, noun: key, glyph: key.slice(0, 3).toUpperCase() });
      const solved = key === 'ladder';
      session.world = worldState({
        entities: session.world.entities.concat([{
          id: session.world.entities.length + 2,
          object: spawned,
          x: HINT.placement.x, y: HINT.placement.y, w: 1, h: 2,
          burning: false, opened: false, cut: false, powered: false, held: false
        }]),
        ticks: session.world.ticks,
        events: ['a ' + body.phrase + ' appears'],
        notebook: session.world.notebook.concat([{ phrase: body.phrase, key: key, approximate: false, x: HINT.placement.x, y: HINT.placement.y }]),
        solved: solved,
        goal: { title: PUZZLE.brief, met: solved, progress: solved ? 'the star is out of the tree' : 'the star is still up there' }
      });
      return json(route, 200, {
        spawned: spawned,
        note: 'a ' + body.phrase + ' appears',
        approximate: false,
        movedSpot: false,
        events: ['a ' + body.phrase + ' appears'],
        hint: solved ? {} : HINT,
        state: session.world
      });
    }

    if (path === '/api/ai/judge') {
      return json(route, 200, {
        verdict: { approved: true, reason: 'a burning rope would part the branch', confidence: 'high' },
        state: worldState({ solved: true, goal: { title: PUZZLE.brief, met: true, progress: 'the star is out of the tree' } }),
        usedAi: true,
        model: 'gpt-4o-mini'
      });
    }

    if (path === '/api/ai/puzzles') {
      return json(route, 201, { spec: AI_SPEC, usedAi: true, model: 'gpt-4o-mini' });
    }

    if (path === '/api/ai/hint') {
      return json(route, 200, { hint: HINT, usedAi: true, model: 'gpt-4o-mini' });
    }

    return json(route, 404, { error: 'not mocked: ' + key });
  });

  return session;
}

/* Fills in the AI settings panel and saves, which is what turns the AI surfaces on. */
async function saveAiSettings(page, key = 'sk-test-not-a-real-key-1234567890') {
  await page.locator('#aiToggle').check();
  await page.locator('#aiBaseUrl').fill('http://localhost:9999/v1');
  await page.locator('#aiApiKey').fill(key);
  await page.locator('#aiModel').fill('gpt-4o-mini');
  await page.locator('#aiSaveButton').click();
  await expect(page.locator('#aiStatus')).toContainText('Ready: gpt-4o-mini at http://localhost:9999/v1');
}

async function openMockedPuzzle(page) {
  await page.goto(APP_URL);
  const card = page.locator('#puzzleList button').first();
  await expect(card).toBeVisible({ timeout: 20000 });
  await card.click();
  await expect(page.locator('#worldSvg')).toHaveAttribute('data-objects', /\d/);
}

/* Collects anything that is a real page failure. A 4xx/5xx response is expected in the error
   tests, and the browser logs "Failed to load resource" for those, so that one is filtered out. */
function watchForCrashes(page) {
  const errors = [];
  page.on('pageerror', (error) => errors.push('pageerror: ' + String(error)));
  page.on('console', (message) => {
    if (message.type() !== 'error') return;
    const text = message.text();
    if (/Failed to load resource|net::ERR/i.test(text)) return;
    errors.push('console: ' + text);
  });
  return errors;
}

/* ================================================================== live stack */

test('live: writing “ladder” puts exactly one ladder in the world', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 900 });
  await page.goto(APP_URL);

  const card = page.locator('#puzzleList button').first();
  await expect(card).toBeVisible({ timeout: 20000 });
  await card.click();
  await expect(page.locator('#worldSvg')).toBeVisible();
  await expect(page.locator('#worldSvg')).toHaveAttribute('aria-label', /cells/);

  await page.locator('#notebookInput').fill('ladder');
  await expect(page.locator('#spawnButton')).toBeEnabled();
  await page.locator('#spawnButton').click();

  // One entity group in the drawing, carrying the key it was drawn for.
  await expect(page.locator('#worldSvg g.entity[data-key="ladder"]')).toHaveCount(1, { timeout: 20000 });
  await expect(page.locator('#worldSvg g.entity[data-key="ladder"] title')).toContainText(/ladder/i);

  // The panel and the log agree with the drawing.
  await expect(page.locator('#entityList .entity[data-key="ladder"]')).toHaveCount(1);
  await expect(page.locator('#entityCount')).toContainText(/object/);
  // The spawn is followed by a short settle step, so the ladder's line is in the log rather than
  // necessarily on top of it.
  await expect(page.locator('#eventLog')).toContainText(/ladder/i);
  await expect(page.locator('#notebookHistory li').first()).toContainText(/ladder/i);
  await expect(page.locator('#spawnNote')).toContainText(/ladder/i);

  // And the glyph label the icon set promises.
  await expect(page.locator('#worldSvg g.entity[data-key="ladder"] text.glyph')).toHaveCount(1);
});

test('live: the player moves and time advances', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 900 });
  await page.goto(APP_URL);
  const card = page.locator('#puzzleList button').first();
  await expect(card).toBeVisible({ timeout: 20000 });
  await card.click();
  await expect(page.locator('#player')).toBeVisible();

  const startX = await page.locator('#worldSvg').getAttribute('data-player-x');
  const startTicks = await page.locator('#tickCount').innerText();

  await page.locator('#btnRight').click();
  await page.locator('#worldSvg').press('ArrowRight');
  await page.locator('#worldSvg').press('Space');
  await page.locator('#btnJump').click();

  // The character is a real element in the layer, and its cell follows the state.
  await expect
    .poll(async () => page.locator('#worldSvg').getAttribute('data-player-x'), { timeout: 20000 })
    .not.toBe(startX);
  const svgX = await page.locator('#worldSvg').getAttribute('data-player-x');
  expect(await page.locator('#player').getAttribute('data-x')).toBe(svgX);

  await page.locator('#btnStep').click();
  await expect.poll(async () => page.locator('#tickCount').innerText(), { timeout: 20000 }).not.toBe(startTicks);
  await expect(page.locator('#tickCount')).toContainText(/tick \d+/);
});

/* ================================================================== mocked: the world, without a backend */

test('mocked: clicking the drawing moves the placement marker and the next spawn lands there', async ({ page }) => {
  const errors = watchForCrashes(page);
  const session = await mockApi(page);
  await openMockedPuzzle(page);

  const marker = page.locator('#placementMarker');
  await expect(marker).toBeVisible();
  const before = await marker.getAttribute('x');

  const box = await page.locator('#worldSvg').boundingBox();
  await page.mouse.click(box.x + box.width * 0.25, box.y + box.height * 0.3);

  await expect.poll(async () => marker.getAttribute('x')).not.toBe(before);
  const cell = {
    x: await marker.getAttribute('x'),
    y: await marker.getAttribute('y')
  };

  await page.locator('#notebookInput').fill('ladder');
  await page.locator('#spawnButton').click();
  await expect.poll(() => session.spawns.length).toBe(1);
  expect(session.spawns[0].x).toBe(Number(cell.x));
  expect(session.spawns[0].y).toBe(Number(cell.y));
  expect(session.spawns[0].phrase).toBe('ladder');
  expect(errors).toEqual([]);
});

test('mocked: a rejected spawn shows the error panel and keeps the page alive', async ({ page }) => {
  const errors = watchForCrashes(page);
  await mockApi(page, {
    'POST /api/worlds/a1b2c3d4e5f6/spawn': { status: 400, body: { error: "the book cannot picture 'florble'" } }
  });
  await openMockedPuzzle(page);

  await page.locator('#notebookInput').fill('florble');
  await page.locator('#spawnButton').click();

  await expect(page.locator('#errorPanel')).toBeVisible();
  await expect(page.locator('#errorMessage')).toContainText("the book cannot picture 'florble'");
  await expect(page.locator('#spawnNote')).toContainText(/did not work/i);
  await expect(page.locator('#spawnNote')).toContainText("the book cannot picture 'florble'");
  await expect(page.locator('#errorRetry')).toBeEnabled();
  await expect(page.locator('#worldSvg')).toBeVisible();
  expect(errors).toEqual([]);
});

test('mocked: a solved state raises the Solved! banner and the world stays playable', async ({ page }) => {
  const errors = watchForCrashes(page);
  await mockApi(page, {
    'POST /api/worlds': {
      status: 200,
      body: { state: worldState({ solved: true, goal: { title: 'Get the star out of the tree.', met: true, progress: 'the star is out of the tree' } }) }
    }
  });
  await openMockedPuzzle(page);

  await expect(page.locator('#solvedBanner')).toBeVisible();
  await expect(page.locator('#solvedBanner')).toContainText('Solved!');
  await expect(page.locator('#goalProgress')).toContainText('the star is out of the tree');
  await expect(page.locator('#goalBanner')).toHaveClass(/is-solved/);

  // Still playable after solving.
  await page.locator('#notebookInput').fill('ladder');
  await page.locator('#spawnButton').click();
  await expect(page.locator('#worldSvg g.entity[data-key="ladder"]')).toHaveCount(1);
  expect(errors).toEqual([]);
});

test('mocked: the page survives a 390px-wide screen without overflowing', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await mockApi(page);
  await openMockedPuzzle(page);

  await expect(page.locator('#worldSvg')).toBeVisible();
  await expect(page.locator('#notebookInput')).toBeVisible();
  await expect(page.locator('#btnLeft')).toBeVisible();
  await expect(page.locator('#hintButton')).toBeVisible();

  const overflow = await page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth + 1);
  expect(overflow).toBe(false);
});

/* ================================================================== mocked: the four choices */

test('mocked: a hint renders four real choices and clicking one spawns it where the hint said', async ({ page }) => {
  const errors = watchForCrashes(page);
  const session = await mockApi(page);
  await openMockedPuzzle(page);

  await page.locator('#hintButton').click();
  const options = page.locator('#hintOptions button.hint-option');
  await expect(options).toHaveCount(4);
  await expect(options).toHaveText([/ladder/i, /cake/i, /pillow/i, /umbrella/i]);
  await expect(page.locator('#hintRationale')).toContainText('one of these will do the job');

  // Nothing on the page may mark which one works: same class, same single data attribute.
  const classes = await options.evaluateAll((els) => els.map((el) => el.className));
  expect(new Set(classes).size).toBe(1);
  const datasets = await options.evaluateAll((els) => els.map((el) => Object.keys(el.dataset).sort().join(',')));
  expect(new Set(datasets)).toEqual(new Set(['phrase']));
  await expect(page.locator('#hintOptions [data-correct]')).toHaveCount(0);

  const phrases = await options.evaluateAll((els) => els.map((el) => el.dataset.phrase));
  expect(phrases).toEqual(['ladder', 'cake', 'pillow', 'umbrella']);

  // A choice is drawn as a small preview plus the phrase.
  await expect(options.first().locator('svg.hint-preview-art')).toHaveCount(1);
  await expect(options.first().locator('.hint-phrase')).toHaveText('ladder');

  await options.first().click();
  await expect.poll(() => session.spawns.length).toBe(1);
  expect(session.spawns[0]).toEqual({ phrase: 'ladder', x: 6, y: 3 });

  // 'ladder' is the one this mocked world can solve, so the honest line says it worked.
  await expect(page.locator('#hintFeedback')).toBeVisible();
  await expect(page.locator('#hintFeedback')).toHaveClass(/is-worked/);
  await expect(page.locator('#solvedBanner')).toBeVisible();
  expect(errors).toEqual([]);
});

test('mocked: a choice that does not reach the goal says so honestly', async ({ page }) => {
  const errors = watchForCrashes(page);
  const session = await mockApi(page);
  await openMockedPuzzle(page);

  await page.locator('#hintButton').click();
  await expect(page.locator('#hintOptions button.hint-option')).toHaveCount(4);
  await page.locator('#hintOptions button.hint-option[data-phrase="pillow"]').click();

  await expect.poll(() => session.spawns.length).toBe(1);
  expect(session.spawns[0].phrase).toBe('pillow');
  await expect(page.locator('#hintFeedback')).toBeVisible();
  await expect(page.locator('#hintFeedback')).toContainText('did not reach the goal — try another of the four');
  await expect(page.locator('#solvedBanner')).toBeHidden();
  expect(errors).toEqual([]);
});

test('mocked: a choice that wins a tick later is reported as working, not as a failure', async ({ page }) => {
  const errors = watchForCrashes(page);
  const session = await mockApi(page, {
    // The choice lands unsolved and settles into a win as soon as the player keeps
    // time moving — exactly like a player climbing a ladder. The line must be
    // amended then, not left saying "try another" under a solved goal.
    'POST /api/worlds/a1b2c3d4e5f6/step': async (route, url, state) => {
      state.world = Object.assign({}, state.world, {
        solved: true,
        goal: { title: PUZZLE.brief, met: true, progress: 'the star is right beside you' }
      });
      return json(route, 200, { events: ['you climb up'], state: state.world });
    }
  });
  await openMockedPuzzle(page);

  await page.locator('#hintButton').click();
  await page.locator('#hintOptions button.hint-option[data-phrase="pillow"]').click();
  await expect.poll(() => session.spawns.length).toBe(1);
  await expect(page.locator('#hintFeedback')).toContainText('did not reach the goal — try another of the four');

  await page.click('#btnStep');
  await expect(page.locator('#solvedBanner')).toBeVisible();
  await expect(page.locator('#hintFeedback')).toHaveClass(/is-worked/);
  await expect(page.locator('#hintFeedback')).toContainText('worked and the goal is met');
  expect(errors).toEqual([]);
});

test('mocked: a choice that no longer fits refreshes the four choices instead of stranding the player', async ({ page }) => {
  const errors = watchForCrashes(page);
  const session = await mockApi(page, {
    // The spot is crowded by the player's own earlier tries.
    'POST /api/worlds/a1b2c3d4e5f6/hint/choose': (route) => json(route, 400, { error: 'a scaffolding needs more room than (5, 8) has' })
  });
  await openMockedPuzzle(page);

  await page.locator('#hintButton').click();
  await expect(page.locator('#hintOptions button.hint-option')).toHaveCount(4);
  const hintsAfterFirst = session.hints;

  await page.locator('#hintOptions button.hint-option[data-phrase="ladder"]').click();
  await expect(page.locator('#hintFeedback')).toContainText('could not be placed at the hint spot');
  // A fresh hint was requested for the world as it now stands.
  await expect.poll(() => session.hints).toBeGreaterThan(hintsAfterFirst);
  await expect(page.locator('#hintOptions button.hint-option')).toHaveCount(4);
  expect(errors).toEqual([]);
});

test('mocked: Test connection proves the settings, or reports why they do not work', async ({ page }) => {
  const errors = watchForCrashes(page);
  const session = await mockApi(page, {
    'POST /api/ai/ping': (route) => json(route, 200, { ok: true, usedAi: true, model: 'gpt-4o-mini', reply: 'ready', latencyMs: 412 })
  });
  await openMockedPuzzle(page);

  // Nothing typed yet: the button says so instead of firing a request.
  await expect(page.locator('#aiTestButton')).toBeDisabled();
  await expect(page.locator('#aiTestResult')).toContainText('Not tested yet');

  await page.locator('#aiToggle').check();
  await page.locator('#aiBaseUrl').fill('http://localhost:9999/v1');
  await page.locator('#aiApiKey').fill('«redacted:sk-…»');
  await page.locator('#aiModel').fill('gpt-4o-mini');
  await page.locator('#aiSaveButton').click();
  await expect(page.locator('#aiTestButton')).toBeEnabled();

  await page.locator('#aiTestButton').click();
  await expect(page.locator('#aiTestResult')).toHaveClass(/is-worked/);
  await expect(page.locator('#aiTestResult')).toContainText('Working: gpt-4o-mini answered "ready" in 412 ms');
  expect(errors).toEqual([]);
});

test('mocked: a failed Test connection is reported word for word', async ({ page }) => {
  const errors = watchForCrashes(page);
  await mockApi(page, {
    'POST /api/ai/ping': (route) => json(route, 502, { error: 'auth (HTTP 401): the AI endpoint refused the key (invalid api key)', usedAi: true })
  });
  await openMockedPuzzle(page);
  await page.locator('#aiToggle').check();
  await page.locator('#aiBaseUrl').fill('http://localhost:9999/v1');
  await page.locator('#aiApiKey').fill('«redacted:sk-…»');
  await page.locator('#aiModel').fill('gpt-4o-mini');
  await page.locator('#aiSaveButton').click();

  await page.locator('#aiTestButton').click();
  await expect(page.locator('#aiTestResult')).toHaveClass(/is-try-again/);
  await expect(page.locator('#aiTestResult')).toContainText('the AI endpoint refused the key');
  expect(errors).toEqual([]);
});

test('mocked: a refused hint shows the server message and no exception', async ({ page }) => {
  const errors = watchForCrashes(page);
  await mockApi(page, {
    'POST /api/worlds/a1b2c3d4e5f6/hint': { status: 400, body: { error: 'the world is already solved, so there is nothing to hint at' } }
  });
  await openMockedPuzzle(page);

  await page.locator('#hintButton').click();
  await expect(page.locator('#hintError')).toBeVisible();
  await expect(page.locator('#hintError')).toContainText('the world is already solved, so there is nothing to hint at');
  await expect(page.locator('#hintOptions button.hint-option')).toHaveCount(0);
  expect(errors).toEqual([]);
});

test('mocked: the AI hint comes back through the same four choices', async ({ page }) => {
  const errors = watchForCrashes(page);
  await mockApi(page);
  await openMockedPuzzle(page);
  await saveAiSettings(page);

  await expect(page.locator('#aiHintButton')).toBeEnabled();
  await page.locator('#aiHintButton').click();

  await expect(page.locator('#hintOptions button.hint-option')).toHaveCount(4);
  await expect(page.locator('#hintRationale')).toContainText('gpt-4o-mini');
  await expect(page.locator('#hintPanel')).toHaveAttribute('data-source', 'ai');
  expect(errors).toEqual([]);
});

/* ================================================================== mocked: the AI surfaces */

test('mocked: saved AI settings report Ready and never show the key in the clear', async ({ page }) => {
  const errors = watchForCrashes(page);
  await mockApi(page);
  const KEY = 'sk-test-not-a-real-key-1234567890';

  await page.goto(APP_URL);
  await expect(page.locator('#aiStatus')).toContainText('AI features are off');
  await saveAiSettings(page, KEY);

  // Only one thing is ever stored, and the field is a password field.
  const stored = await page.evaluate(() => {
    const keys = Object.keys(window.localStorage);
    const raw = window.localStorage.getItem('scribblebox.ai');
    return { keys: keys, value: raw ? JSON.parse(raw) : null };
  });
  expect(stored.keys).toEqual(['scribblebox.ai']);
  expect(stored.value.apiKey).toBe(KEY);
  expect(stored.value.enabled).toBe(true);

  await expect(page.locator('#aiApiKey')).toHaveAttribute('type', 'password');
  const leaked = await page.evaluate((secret) => ({
    text: document.body.innerText.includes(secret),
    markup: document.body.innerHTML.includes(secret)
  }), KEY);
  expect(leaked.text).toBe(false);
  expect(leaked.markup).toBe(false);

  // Revealing is a deliberate act, and it goes back to hidden.
  await page.locator('#aiKeyReveal').click();
  await expect(page.locator('#aiApiKey')).toHaveAttribute('type', 'text');
  await page.locator('#aiKeyReveal').click();
  await expect(page.locator('#aiApiKey')).toHaveAttribute('type', 'password');

  await page.locator('#aiForgetButton').click();
  await expect(page.locator('#aiStatus')).toContainText('AI features are off');
  expect(await page.evaluate(() => window.localStorage.length)).toBe(0);
  expect(errors).toEqual([]);
});

test('mocked: an approved verdict renders the reason, the model and the solved world', async ({ page }) => {
  const errors = watchForCrashes(page);
  await mockApi(page);
  await openMockedPuzzle(page);
  await saveAiSettings(page);

  await expect(page.locator('#aiJudgeButton')).toBeEnabled();
  await page.locator('#aiJudgeQuestion').fill('would a burning rope part the branch?');
  await page.locator('#aiJudgeButton').click();

  await expect(page.locator('#aiVerdict')).toHaveAttribute('data-approved', 'true');
  await expect(page.locator('#aiVerdict')).toContainText('a burning rope would part the branch');
  await expect(page.locator('#aiVerdict')).toContainText('gpt-4o-mini');
  await expect(page.locator('#solvedBanner')).toBeVisible();
  await expect(page.locator('#aiJudgeError')).toBeHidden();
  expect(errors).toEqual([]);
});

test('mocked: a 502 from the judge is shown word for word', async ({ page }) => {
  const errors = watchForCrashes(page);
  await mockApi(page, {
    'POST /api/ai/judge': { status: 502, body: { error: 'the AI endpoint refused the key (401)', usedAi: true } }
  });
  await openMockedPuzzle(page);
  await saveAiSettings(page);

  await page.locator('#aiJudgeQuestion').fill('would a burning rope part the branch?');
  await page.locator('#aiJudgeButton').click();

  await expect(page.locator('#aiJudgeError')).toBeVisible();
  await expect(page.locator('#aiJudgeError')).toHaveText('the AI endpoint refused the key (401)');
  await expect(page.locator('#aiVerdict')).not.toHaveAttribute('data-approved', 'true');
  await expect(page.locator('#solvedBanner')).toBeHidden();
  expect(errors).toEqual([]);
});

test('mocked: with AI off the judge button is disabled and says why', async ({ page }) => {
  await mockApi(page);
  await page.goto(APP_URL);
  await expect(page.locator('#aiToggle')).not.toBeChecked();
  await expect(page.locator('#aiJudgeButton')).toBeDisabled();
  await expect(page.locator('#aiVerdict')).toContainText('AI features are off');
  await expect(page.locator('#aiHintButton')).toBeDisabled();
  await expect(page.locator('#aiPuzzleButton')).toBeDisabled();

  // Turning it on without a key is not a working setup, and the page says so instead of pretending.
  await page.locator('#aiToggle').check();
  await expect(page.locator('#aiJudgeButton')).toBeDisabled();
  await expect(page.locator('#aiStatus')).toContainText('Not configured');
});

test('mocked: a generated puzzle opens as a world, badged as AI-made', async ({ page }) => {
  const errors = watchForCrashes(page);
  const session = await mockApi(page);
  await page.goto(APP_URL);
  await expect(page.locator('#aiPuzzleButton')).toBeDisabled();

  await saveAiSettings(page, 'sk-test-not-a-real-key-abcdefghij');
  await page.locator('#aiPuzzleTheme').fill('a lighthouse in a storm');
  await page.locator('#aiPuzzleGoalKind').selectOption('chest');
  await page.locator('#aiPuzzleButton').click();

  await expect(page.locator('#aiWorldBadge')).toBeVisible();
  await expect(page.locator('#aiWorldBadge')).toContainText('AI-made');
  await expect(page.locator('#aiWorldBadge')).toContainText('gpt-4o-mini');
  await expect(page.locator('#worldSvg')).toHaveAttribute('data-ai', 'true');
  await expect(page.locator('#worldSvg')).toHaveAttribute('aria-label', /lighthouse/i);
  await expect(page.locator('#goalTitle')).toContainText('Reach the lamp room');
  expect(session.aiWorlds).toBe(1);
  expect(errors).toEqual([]);
});

test('mocked: a failed generation is reported without throwing', async ({ page }) => {
  const errors = watchForCrashes(page);
  await mockApi(page, {
    'POST /api/ai/puzzles': { status: 502, body: { error: 'the AI endpoint refused the key (401)', usedAi: true } }
  });
  await page.goto(APP_URL);
  await saveAiSettings(page, 'sk-test-not-a-real-key-abcdefghij');

  await page.locator('#aiPuzzleTheme').fill('a lighthouse in a storm');
  await page.locator('#aiPuzzleButton').click();

  await expect(page.locator('#aiPuzzleError')).toBeVisible();
  await expect(page.locator('#aiPuzzleError')).toHaveText('the AI endpoint refused the key (401)');
  await expect(page.locator('#aiPuzzleStatus')).toContainText(/no puzzle/i);
  expect(errors).toEqual([]);
});
