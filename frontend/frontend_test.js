/* Node-only static checks for the Scribblebox frontend.
 * No browser, no network: every assertion reads a file from disk (or evaluates a slice of app.js
 * with vm). Run with: npm test   (i.e. node frontend_test.js)
 */
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");

const read = (relative) => fs.readFileSync(path.join(__dirname, relative), "utf8");

const html = read("index.html");
const js = read("app.js");
const css = read("styles.css");
const nginx = read("nginx.conf");
const dockerfile = read("Dockerfile");
const pkg = JSON.parse(read("package.json"));
const compose = read(path.join("..", "docker-compose.yml"));

/* ---------------------------------------------------------------- syntax */

assert.doesNotThrow(() => new vm.Script(js), "app.js should parse");

/* ---------------------------------------------------------------- element ids */

// Every id app.js reaches for must exist in index.html: a missing id is a dead control.
const referencedIds = new Set();
for (const match of js.matchAll(/getElementById\(\s*["'`]([^"'`]+)["'`]\s*\)/g)) referencedIds.add(match[1]);
for (const match of js.matchAll(/\$\(\s*["'`]([^"'`]+)["'`]\s*\)/g)) referencedIds.add(match[1]);
assert.ok(
  referencedIds.size >= 65,
  `app.js should drive the whole game, but only ${referencedIds.size} element ids were referenced`
);

const missingIds = [...referencedIds].filter((id) => !new RegExp(`id=["']${id}["']`).test(html));
assert.deepEqual(missingIds, [], `index.html is missing ids referenced by app.js: ${missingIds.join(", ")}`);

for (const id of [
  // puzzle picker · world · notebook · goal · log · entities
  "puzzleList", "puzzleBrief", "puzzleHint", "puzzleHintButton", "worldSvg", "worldStatus",
  "tickCount", "notebookForm", "notebookInput", "notebookSuggestions", "spawnButton", "spawnNote",
  "notebookHistory", "eventLog", "entityList", "entityCount", "goalBanner", "goalTitle",
  "goalProgress", "solvedBanner", "btnLeft", "btnRight", "btnJump", "btnTake", "btnDrop",
  "btnStep", "btnReset", "newPuzzleButton", "errorPanel", "errorMessage", "errorRetry", "placementHint",
  "terrainLayer", "entityLayer", "playerLayer", "placementMarker",
  // four choices
  "hintButton", "hintPanel", "hintRationale", "hintOptions", "hintFeedback", "hintError",
  // AI settings · judge · puzzle generator
  "aiSettings", "aiToggle", "aiSettingsForm", "aiBaseUrl", "aiApiKey", "aiKeyReveal", "aiModel",
  "aiSaveButton", "aiForgetButton", "aiStatus", "aiSettingsNote", "aiHintButton",
  "aiJudgePanel", "aiJudgeQuestion", "aiJudgeButton", "aiVerdict", "aiJudgeError",
  "aiPuzzlePanel", "aiPuzzleTheme", "aiPuzzleGoalKind", "aiPuzzleButton", "aiPuzzleStatus", "aiPuzzleError",
  "aiWorldBadge"
]) {
  assert.match(html, new RegExp(`id=["']${id}["']`), `missing required UI element #${id}`);
}

// Every aria/for reference must point at an element that really exists, and ids must be unique.
const referenceTargets = [...html.matchAll(/(?:aria-controls|aria-labelledby|aria-describedby|for)="([^"]+)"/g)].map((match) => match[1]);
const danglingReferences = referenceTargets.filter((id) => !new RegExp(`id=["']${id}["']`).test(html));
assert.deepEqual(danglingReferences, [], `index.html points at ids that do not exist: ${danglingReferences.join(", ")}`);

const htmlIds = [...html.matchAll(/id="([^"]+)"/g)].map((match) => match[1]);
assert.equal(new Set(htmlIds).size, htmlIds.length, "index.html must not repeat an element id");

/* ---------------------------------------------------------------- inline SVG, not canvas */

assert.doesNotMatch(html, /<canvas/i, "the world must be inline SVG, not a canvas");
assert.doesNotMatch(js, /getContext|canvas/i, "app.js must not draw on a canvas any more");
assert.doesNotMatch(html, /worldCanvas/, "the old canvas id must be gone");
assert.match(html, /<svg id="worldSvg"/, "the world must be one inline <svg>");
assert.match(html, /id="worldSvg"[^>]*viewBox="0 0 28 16"/, "the SVG needs a viewBox to scale from");
assert.match(html, /id="worldSvg"[^>]*preserveAspectRatio="xMidYMid meet"/, "the SVG must keep its aspect ratio");
assert.match(html, /id="worldSvg"[^>]*role="img"/, "the SVG must be announced as an image");
assert.match(html, /id="worldSvg"[^>]*tabindex="0"/, "the SVG must be keyboard-focusable");
assert.match(html, /id="worldSvg"[^>]*aria-label="[^"]+"/, "the SVG must describe itself");
assert.match(js, /svg\.setAttribute\("aria-label", worldLabel\(world\)\)/, "the label must be refreshed with the world");
assert.match(js, /return title \+ ", " \+ num\(world\.width, 1\) \+ " by " \+ num\(world\.height, 1\) \+ " cells, "/, "the label reads “title, W by H cells, N objects”");
assert.match(css, /#worldSvg\s*\{[^}]*width:\s*100%/, "the drawing must scale with CSS");
assert.match(css, /#worldSvg\s*\{[^}]*height:\s*auto/, "the drawing must keep its own height");

// Layer order: terrain behind, then entities, then the player, with the marker above them all.
const layerOrder = ["terrainLayer", "entityLayer", "playerLayer", "placementMarker"]
  .map((id) => html.indexOf(`id="${id}"`));
assert.ok(layerOrder.every((position) => position !== -1), "all four world layers must exist");
assert.deepEqual(layerOrder.slice().sort((a, b) => a - b), layerOrder, "terrain, then entities, then the player, then the marker");
assert.match(js, /\$\("terrainLayer"\)\.innerHTML/, "the terrain must be filled from the state");
assert.match(js, /\$\("entityLayer"\)\.innerHTML = entities\.map\(entityArt\)\.join\(""\)/, "one group per entity");
assert.match(js, /\$\("playerLayer"\)\.innerHTML = playerArt\(/, "the player has its own layer");

/* ---------------------------------------------------------------- the twenty shapes */

const shapeStart = js.indexOf("const SHAPE_ART = {");
assert.notEqual(shapeStart, -1, "app.js must define one SHAPE_ART lookup");
let depth = 0;
let shapeEnd = -1;
for (let index = js.indexOf("{", shapeStart); index < js.length; index += 1) {
  const character = js[index];
  if (character === "{") depth += 1;
  else if (character === "}") {
    depth -= 1;
    if (depth === 0) { shapeEnd = index + 1; break; }
  }
}
assert.ok(shapeEnd > shapeStart, "the SHAPE_ART object must be complete");

const inkStart = js.indexOf('const INK = "#26323d";');
assert.ok(inkStart !== -1 && inkStart < shapeStart, "the shape map must sit inside one self-contained block");
const shapeSource = js.slice(inkStart, shapeEnd);

const SHAPES = [
  "box", "ladder", "rope", "plank", "blob", "circle", "star", "tree", "flame", "key",
  "tool", "animal", "person", "bottle", "book", "vehicle", "chest", "candle", "flag", "machine"
];

// The slice is evaluated for real, so every shape is proved to return non-empty SVG.
const sandbox = vm.createContext({});
const shapeValue = new vm.Script(shapeSource + "\n;({ SHAPE_ART: SHAPE_ART, INK: INK, STROKE: STROKE });").runInContext(sandbox);
const SHAPE_ART = shapeValue.SHAPE_ART;
assert.equal(shapeValue.INK, "#26323d", "one outline colour for the whole icon set");
assert.equal(shapeValue.STROKE, 0.05, "one outline width for the whole icon set");
assert.equal(Object.keys(SHAPE_ART).length, 20, "the shape map must cover exactly the twenty pinned shapes");

for (const shape of SHAPES) {
  const draw = SHAPE_ART[shape];
  assert.equal(typeof draw, "function", `SHAPE_ART.${shape} must be a drawing function`);
  for (const size of [[1, 1], [1, 2], [2, 1], [2, 2]]) {
    const fragment = draw(size[0], size[1], "#c98a3c", { opened: true, burning: true });
    assert.equal(typeof fragment, "string", `SHAPE_ART.${shape} must return a string`);
    assert.ok(fragment.length > 20, `SHAPE_ART.${shape} must draw a real shape at ${size.join("x")}`);
    assert.match(fragment, /<(rect|circle|ellipse|path|polygon|line|g)\b/, `SHAPE_ART.${shape} must return SVG elements`);
    assert.equal(
      draw(size[0], size[1], "#c98a3c", { opened: true, burning: true }),
      fragment,
      `SHAPE_ART.${shape} must be deterministic: the same state has to draw the same picture`
    );
  }
}

assert.equal(SHAPE_ART["sun-please-do-not-exist"], undefined, "the map must not pretend to know an unknown shape");
assert.equal(
  (SHAPE_ART["unknown-shape"] || SHAPE_ART.blob)(1, 1, "#ffffff", {}),
  SHAPE_ART.blob(1, 1, "#ffffff", {}),
  "an unknown shape must fall back to the blob"
);
assert.match(js, /const art = SHAPE_ART\[shape\] \|\| SHAPE_ART\.blob;/, "the renderer must use the blob fallback");
assert.match(js, /Object\.prototype\.hasOwnProperty\.call\(SHAPE_ART, object\.shape\)/, "an unknown shape name must be recognised as unknown");

// Named details the tests can rely on.
assert.match(SHAPE_ART.ladder(1, 2, "#c98a3c", {}), /<rect/, "the ladder must have rails and rungs");
assert.match(SHAPE_ART.rope(1, 1, "#c98a3c", {}), /q0\.2 -0\.16/, "the rope must be a smooth squiggle");
assert.match(SHAPE_ART.star(1, 1, "#c98a3c", {}), /<polygon points="/, "the star must be a real polygon");
const keyArt = SHAPE_ART.key(1, 1, "#c98a3c", {});
assert.match(keyArt, /<circle/, "the key must have a round bow");
assert.ok((keyArt.match(/<rect/g) || []).length >= 4, "the key must have a shaft and three teeth");
assert.match(SHAPE_ART.chest(1, 1, "#c98a3c", { opened: true }), /<path/, "an opened chest must draw its lid open");
assert.notEqual(SHAPE_ART.chest(1, 1, "#c98a3c", { opened: true }), SHAPE_ART.chest(1, 1, "#c98a3c", { opened: false }), "an open chest must differ from a shut one");
assert.equal(SHAPE_ART.candle(1, 1, "#c98a3c", { burning: false }).includes("flame-lick"), false, "an unlit candle has no flame");
assert.match(SHAPE_ART.candle(1, 1, "#c98a3c", { burning: true }), /flame-lick/, "a lit candle has a flame");
assert.match(SHAPE_ART.vehicle(1, 1, "#c98a3c", {}), /<circle[\s\S]*<circle/, "the vehicle must have two wheels");
assert.match(SHAPE_ART.machine(1, 1, "#c98a3c", {}), /<circle/, "the machine must have a dial");
assert.match(SHAPE_ART.flag(1, 1, "#c98a3c", {}), /<path/, "the flag must be a pole with a pennant");
for (const withFace of ["animal", "person"]) {
  const fragment = SHAPE_ART[withFace](2, 2, "#c98a3c", {});
  assert.match(fragment, /q /, `${withFace} needs a small smile`);
  assert.ok((fragment.match(/<circle/g) || []).length >= 2, `${withFace} needs two dot eyes`);
}

/* ---------------------------------------------------------------- one clean style, fully deterministic */

// One consistent outline, taken from the entity group rather than repeated per shape.
assert.match(js, /const INK = "#26323d";/, "the outline colour must be the one shared ink");
assert.match(js, /const STROKE = 0\.05;/, "the outline width must be the one shared 0.05");
assert.match(js, /stroke="' \+ INK \+ '" stroke-width="' \+ STROKE \+ '"/, "the outline must be applied from the shared ink and width");
assert.match(js, /stroke-linejoin="round" stroke-linecap="round"/, "outlines must be rounded, like any tidy icon set");

// Nothing that would make the drawing gritty or unstable.
for (const forbidden of ["Math.random", "Date.now", "performance", "new Date", "requestAnimationFrame", "rotate(", "Gradient", "shadowBlur", "drop-shadow", "non-scaling-stroke", "<filter"]) {
  assert.equal(js.includes(forbidden), false, `app.js must not use ${forbidden}: the drawing must be clean and deterministic`);
}
assert.equal(html.includes("Math.random"), false, "index.html must not rely on a random number");
assert.doesNotMatch(css, /filter:|drop-shadow/i, "the stylesheet must not drop shadows on the drawing");
assert.match(js, /transform="translate\(' \+ round2\(x\) \+ ' ' \+ round2\(y\) \+ '\)"/, "an entity is placed with a plain translate, never a jitter");

/* ---------------------------------------------------------------- entities, player and terrain */

assert.match(js, /' data-entity-id="' \+ escapeXml\(num\(entity\.id, 0\)\)/, "each entity group must carry its id");
assert.match(js, /' data-key="' \+ escapeXml\(object\.key \|\| ""\)/, "each entity group must carry its key");
assert.match(js, /' data-shape="' \+ escapeXml\(shape\)/, "each entity group must carry the shape it was drawn with");
assert.match(js, /' data-tags="' \+ escapeXml\(tags\.join\(","\)\)/, "each entity group must carry its tags");
for (const flag of ["burning", "opened", "cut", "powered", "held"]) {
  assert.ok(js.includes(`' data-${flag}="' + (entity.${flag} ? "true" : "false")`), `each entity group must carry data-${flag}`);
}
assert.match(js, /'<title>' \+ escapeXml\(title\) \+ '<\/title>'/, "each entity needs a <title> so hover explains what it is");
assert.match(js, /name \+ " — " \+/, "the hover title must start with the object's name");
assert.match(js, /font-size="0\.45" text-anchor="middle"/, "the glyph must be a small centred label");
assert.match(js, /font-family="monospace"/, "no font may be loaded: at most a generic family");
assert.match(js, /escapeXml\(glyph\)/, "the glyph must be escaped before it goes into the DOM");
assert.match(js, /entity\.burning\) \{[\s\S]{0,400}flame-lick/, "a burning entity must get flame art");
assert.match(css, /\.is-burning \.flame-lick[\s\S]{0,120}animation:/, "the flames must have a slow animation");
assert.match(css, /\.is-powered \.glow[\s\S]{0,120}animation-name: pulse/, "a powered machine must pulse its glow");
assert.match(js, /entity\.cut \? ' opacity="0\.5"'/, "a cut entity must be dimmed");
assert.match(js, /entity\.cut\) \{[\s\S]{0,300}stroke-dasharray/, "a cut entity must get a dashed outline");

assert.match(js, /'<g id="player" data-x="' \+ round2\(x\) \+ '" data-y="' \+ round2\(y\)/, "the player group must publish its cell");
assert.match(js, /'<g transform="scale\(0\.94 0\.94\)' \+ flip/, "the character must have its own inner group to flip");
assert.match(js, /const flip = left \? ' scale\(-1 1\) translate\(-1 0\)' : "";/, "walking left must flip the character with scale(-1 1)");
assert.match(js, /state\.facing === "left"|state\.facing = "left"/, "the facing must follow the player's own actions");
assert.match(js, /\$\("playerLayer"\)\.innerHTML = playerArt\(world\.player \|\| \{ x: 0, y: 0, onGround: true \}, entities\)/, "the player is drawn from state.player every render");
assert.match(js, /round head|#f4d3ac/, "the character must be a friendly little figure");
assert.match(js, /notebook|fdf6e3/, "the character must be holding a notebook");

for (const terrain of ["#c98a4b", "#7cc4f2", "#9a6b3a", "#dfe4ea", "#c9c6c1"]) {
  assert.ok(js.includes(terrain), `terrain colour ${terrain} must be drawn`);
}
assert.match(js, /if \(!value\) return "";/, "air must stay empty");
assert.match(js, /function grassTicks\(/, "the ground needs evenly spaced grass ticks");
assert.match(js, /q0\.19 -0\.08 0\.38 0/, "water needs tidy wave lines");
assert.match(js, /const same = above === value;/, "terrain must consult the cell above so a mass reads as one shape");

/* ---------------------------------------------------------------- click to place */

assert.match(js, /addEventListener\("click", onSvgClick\)/, "a click on the drawing must choose the spawn cell");
assert.match(js, /getBoundingClientRect\(\)/, "the click must be measured against the rendered box");
assert.match(js, /svg\.viewBox\.baseVal/, "the click must be mapped through the viewBox, not a canvas");
assert.match(js, /const scale = Math\.min\(rect\.width \/ vw, rect\.height \/ vh\)/, "the letterboxing must be accounted for");
assert.match(js, /state\.placement = target/, "the click must store the chosen cell");
assert.match(js, /marker\.setAttribute\("x", target\.x\)/, "the marker rect must move to the chosen cell");
assert.match(js, /marker\.removeAttribute\("hidden"\);/, "the marker must be shown for the open world");
assert.match(css, /#placementMarker\[hidden\][\s\S]{0,40}display:\s*none/, "the marker's hidden attribute must actually hide it");
assert.match(html, /<rect id="placementMarker"/, "the marker must be a real rect in the markup");
assert.match(js, /source: "sky"/, "there must be a default spawn cell above the player");
assert.match(js, /num\(player\.y, 0\) - 3/, "the default spot is above the player — drop from the sky");
for (const needed of ["id=\"btnDropFromSky\"", "id=\"placementHint\""]) {
  assert.ok(html.includes(needed), `the placement UI must include ${needed}`);
}

/* ---------------------------------------------------------------- the notebook + autocomplete */

assert.match(js, /"\/api\/words\?q=" \+ encodeURIComponent\(asked\) \+ "&limit=" \+ WORDS_LIMIT/, "autocomplete must ask GET /api/words?q=&limit=");
assert.match(js, /body: \{ phrase: text, x: target\.x, y: target\.y \}/, "a spawn must send phrase, x and y");
for (const key of ["ArrowDown", "ArrowUp", "Escape", "Enter"]) {
  assert.ok(js.includes(key), `the suggestion list must be keyboard navigable (${key})`);
}
assert.match(js, /role", "option"/, "each suggestion must be a real option");
assert.match(js, /addEventListener\("mousedown", \(event\) => \{/, "suggestions must be clickable");
assert.match(js, /aria-activedescendant/, "the highlighted suggestion must be announced");
assert.match(js, /state\.notebook = state\.notebook\.concat/, "the notebook history must record every phrase");
assert.match(js, /The book has never heard of that, so it improvised\./, "an approximate word must be admitted");
assert.match(js, /The book knew that word\./, "a known word must be reported as known");
assert.match(js, /result\.data\.note/, "the API's own note must be shown after a spawn");

/* ---------------------------------------------------------------- controls, time, goal, log */

assert.match(js, /const ACTIONS = \["left", "right", "jump", "take", "drop"\];/, "the five actions must be declared");
for (const action of ["left", "right", "jump", "take", "drop"]) {
  assert.ok(js.includes(`doAction("${action}")`), `the ${action} action must be wired`);
}
for (const key of ["ArrowLeft", "ArrowRight", "Spacebar", "case \"t\"", "case \"d\""]) {
  assert.ok(js.includes(key), `the keyboard must drive the player (${key})`);
}
assert.match(js, /const STEP_TICKS = 4;/, "advance time must be four ticks");
assert.match(js, /const SETTLE_TICKS = 2;/, "a spawn must settle with a two-tick step");
assert.match(js, /ticks: STEP_TICKS/, "the advance-time control must send four ticks");
assert.match(js, /ticks: SETTLE_TICKS/, "the spawn must send the settling step");
assert.match(js, /svg\.dataset\.ticks = String\(num\(world\.ticks, 0\)\)/, "the tick count must be published and shown");
assert.match(js, /svg\.dataset\.playerX/, "the player position must be published too");
assert.match(js, /ROUTES\.worlds \+ "\/" \+ encodeURIComponent\(state\.worldId\) \+ "\/act"/, "an action must go to the open world");
assert.match(js, /ROUTES\.worlds \+ "\/" \+ encodeURIComponent\(state\.worldId\) \+ "\/step"/, "time must advance on the open world");
assert.match(js, /ROUTES\.worlds \+ "\/" \+ encodeURIComponent\(state\.worldId\) \+ "\/reset"/, "a reset must go to the open world");
assert.match(js, /const result = await request\(ROUTES\.worlds, \{ method: "POST", body: \{ puzzleId: puzzle\.id \} \}\)/, "picking a puzzle must open a world with its puzzleId");
assert.match(js, /goal\.progress/, "the goal banner must show the progress line");
assert.match(js, /world\.solved === true \|\| goal\.met === true/, "the solved state must come from state.solved (or a met goal)");
assert.match(js, /solvedBanner"\)\.hidden = !solved/, "Solved! must be shown prominently and hidden again on reset");
assert.match(js, /incoming\.slice\(\)\.reverse\(\)\.concat\(state\.events\)/, "the newest events go in front of the log");
assert.match(js, /fromState\.slice\(\)\.reverse\(\)\.forEach/, "the state's own events are also shown newest first");
assert.match(js, /item\.dataset\.key = String\(object\.key \|\| ""\)/, "the entity panel must show each key");
assert.match(js, /tagList\.forEach/, "the entity panel must list the tags");
for (const flag of ["burning", "opened", "cut", "powered", "held"]) {
  assert.ok(js.includes(`item.dataset.${flag}`), `the entity panel must show the ${flag} flag`);
}
assert.match(js, /entityCount"\)\.textContent/, "the entity count must be rendered");
assert.match(js, /addEventListener\("click", resetPuzzle\)/, "there must be a reset button");
assert.match(js, /errorMessage"\)\.textContent/, "a failed request must reach the error panel");
assert.match(js, /errorRetry"\)\.addEventListener\("click"/, "the error panel must offer Retry");
assert.match(js, /Cannot reach the Scribblebox service/, "an unreachable API must be reported as such");
assert.match(js, /await response\.text\(\)/, "the transport must read the raw body to tell network, parse and HTTP failures apart");
assert.match(js, /data\.error/, "HTTP errors must surface the API's own {\"error\": \"…\"} message");
assert.match(js, /queue = queue\.then\(run, run\);/, "every call runs in order through one queue, so state can never go backwards");

/* ---------------------------------------------------------------- four choices (hint) */

assert.match(js, /ROUTES\.worlds \+ "\/" \+ encodeURIComponent\(state\.worldId\) \+ "\/hint", \{ method: "POST", body: \{\} \}/, "the hint must POST to /api/worlds/{id}/hint with no body");
assert.match(js, /state\.hint = \{/, "the four choices must be kept for the click to use");
assert.match(js, /function hintOptionButton\(phrase\)/, "each choice must be built as its own element");
assert.match(js, /button\.className = "hint-option";/, "each choice must be a real button.hint-option");
assert.match(js, /button\.dataset\.phrase = phrase;/, "each choice must carry its phrase");
assert.match(js, /button\.addEventListener\("click", \(\) => chooseHintOption\(phrase\)\)/, "pressing a choice must be wired");
assert.match(js, /options\.forEach\(\(phrase\) => list\.appendChild\(hintOptionButton\(phrase\)\)\)/, "the four choices must be appended as buttons");
assert.match(js, /const options = hint\.options\.slice\(0, 4\)\.map\(String\);/, "a hint offers four choices");
assert.match(js, /ROUTES\.worlds \+ "\/" \+ encodeURIComponent\(state\.worldId\) \+ "\/hint\/choose"/, "a choice must be placed by the server so it can re-prove the spot when it is pressed");
assert.match(js, /state\.hintNotice = line;/, "the line about a choice must survive the fresh four that follow it");
assert.match(js, /hintChoiceFeedback\(text\);/, "a chosen option must report honestly afterwards");
assert.match(js, /did not reach the goal — try another of the four\./, "an unmet goal must say to try another of the four");
assert.match(js, /worked and the goal is met — the goal banner above has the rest\./, "a met goal must say the choice worked");
assert.match(js, /const solved = world\.solved === true \|\| goal\.met === true;/, "the feedback must be read from the world's own state");
assert.match(js, /function hintPreviewSvg\(phrase, knowledge, shape, color\)/, "each choice gets a small preview drawing");
assert.match(js, /const art = known && shape && SHAPE_ART\[shape\]/, "a known word is drawn with the shape the dictionary reports");
assert.match(js, /function refineHintPreview\(phrase\)/, "the preview must be refined without blocking the button");
assert.match(js, /options\.forEach\(\(phrase\) => refineHintPreview\(phrase\)\)/, "previews are refined after the buttons are already on screen");
assert.match(js, /preview\.dataset\.preview = known \? "known" : "improvised";/, "the preview must say whether the word was known");
assert.match(js, /setHintError\(result\.error\)/, "a refused hint must show the server's message verbatim");
assert.match(js, /function askForHint\(\)/, "the hint button needs its own request");

// The document must never hint at which option is the right one.
assert.doesNotMatch(js, /data-correct/, "no option may be marked correct in the DOM");
assert.doesNotMatch(html, /data-correct/, "no option may be marked correct in the markup either");
assert.doesNotMatch(js, /correct/i, "app.js must not contain the word “correct” at all: the player finds out by trying");
assert.doesNotMatch(html, /correct/i, "index.html must not contain the word “correct” either");
const optionBody = functionBody("hintOptionButton");
assert.equal(
  (optionBody.match(/button\.dataset\.\w+ =/g) || []).join(","),
  "button.dataset.phrase =",
  "an option carries its phrase and nothing else — no marker the UI could have been handed"
);
assert.match(optionBody, /button\.addEventListener\("click", \(\) => chooseHintOption\(phrase\)\)/, "the click is the only way an option does anything");
assert.match(js, /hint: "\/api\/ai\/hint"/, "the AI hint must POST to /api/ai/hint");
assert.match(js, /body: \{ baseUrl: ai\.baseUrl, apiKey: ai\.apiKey, model: ai\.model, worldId: state\.worldId \}/, "the AI hint must send the settings and the world id");
assert.match(js, /renderHint\(result\.data \? result\.data\.hint : null, \{[\s\S]{0,200}\? "engine" : "ai"/, "an AI hint must render through the same options path");
assert.match(js, /renderHint\(result\.data \? result\.data\.hint : null, \{ source: "engine" \}\)/, "the engine's own four choices go through that same path too");
assert.match(js, /const usable = ai\.enabled && aiConfigured\(\) && !!state\.world;\s+\$\("aiHintButton"\)\.disabled = !usable;/, "the AI hint button must be disabled when AI is off");
assert.match(js, /AI is off, so this stays disabled/, "and it must say why");

/* ---------------------------------------------------------------- accessibility + responsive */

assert.match(html, /<button id="btnLeft"[^>]*aria-label="[^"]+"/, "controls must be real buttons with labels");
for (const id of ["btnLeft", "btnRight", "btnJump", "btnTake", "btnDrop", "btnStep", "btnReset", "spawnButton", "errorRetry", "hintButton", "aiHintButton", "aiSaveButton", "aiKeyReveal"]) {
  assert.match(html, new RegExp(`<button id="${id}"[^>]*aria-label="`), `#${id} must be a real button with an aria label`);
}
assert.match(html, /id="eventLog"[^>]*role="log"/, "the event log must be a live log");
assert.match(html, /id="worldStatus"[^>]*role="status"/, "the world status must be a live status");
assert.match(html, /id="solvedBanner"[^>]*role="status"/, "the solved banner must be announced");
assert.match(html, /id="solvedBanner"[^>]*hidden/, "Solved! starts hidden");
assert.match(html, /id="hintError"[^>]*role="alert"/, "a refused hint must be announced");
assert.match(html, /id="hintOptions"[^>]*role="group"/, "the four choices must be grouped and labelled");
assert.match(css, /:focus-visible/, "keyboard focus must stay visible");
assert.match(css, /@media\s*\(max-width:\s*430px\)/, "the layout must be tuned for 390px-wide phones");
assert.match(css, /@media\s*\(max-width:\s*960px\)/, "the layout must collapse for tablets and phones");
assert.match(css, /\.controls\s*\{[^}]*flex-wrap:\s*wrap/, "the controls must wrap");

/* ---------------------------------------------------------------- no external assets at runtime */

const externalRefs = [...html.matchAll(/(?:src|href)\s*=\s*["']([^"']+)["']/gi)]
  .map((match) => match[1])
  .filter((url) => /^[a-z][a-z0-9+.-]*:\/\//i.test(url) || url.startsWith("//"));
assert.deepEqual(externalRefs, [], "nothing may load from another origin: no CDN, no remote font, no remote image");

assert.doesNotMatch(html, /<img\b/i, "no image files or remote emoji images");
assert.doesNotMatch(html, /<link[^>]*rel="preconnect"/i, "no preconnect to any other host");
assert.match(html, /<script src="app\.js"><\/script>/, "the only script must be local app.js");
assert.match(html, /<link rel="stylesheet" href="styles\.css">/, "the only stylesheet must be local styles.css");
assert.doesNotMatch(css, /@import|url\(\s*["']?https?:/i, "the stylesheet must not import a remote font or image");

// The single absolute URL is the AI base-URL placeholder — a hint for the player, never a fetch target.
const htmlUrls = [...html.matchAll(/https?:\/\/[A-Za-z0-9./-]+/g)].map((match) => match[0]);
assert.deepEqual(htmlUrls, ["https://api.openai.com/v1"], "the only absolute URL in index.html must be the AI base-URL placeholder");
assert.match(html, /id="aiBaseUrl"[^>]*placeholder="https:\/\/api\.openai\.com\/v1"/, "that placeholder belongs to #aiBaseUrl");
const jsUrls = [...js.matchAll(/https?:\/\/[A-Za-z0-9-]+(?:\.[A-Za-z0-9-]+)+/g)].map((match) => match[0]);
assert.deepEqual(jsUrls, [], "app.js must never name another host: it only ever calls its own /api routes");

/* ---------------------------------------------------------------- AI: the key, and where it may go */

assert.match(js, /const AI_STORAGE_KEY = "scribblebox\.ai";/, "the AI settings key must be exactly scribblebox.ai");
assert.match(html, /id="aiApiKey"[^>]*type="password"/, "#aiApiKey must be a password field in the markup");
assert.match(js, /\$\("aiApiKey"\)\.type = "password"/, "the field must be put back to type=password when settings load");
assert.match(js, /field\.type = shown \? "password" : "text";/, "the reveal control is the only thing that can show the key");
const plainTextSwitches = [...js.matchAll(/\.type = [^;]*"text"/g)].length;
assert.equal(plainTextSwitches, 1, "the key field may only become plain text in the reveal control");
assert.match(js, /function toggleKeyReveal\(/, "the reveal must be its own control");
assert.match(html, /id="aiKeyReveal"[^>]*aria-pressed="false"/, "the reveal button must announce its state");

const storageLiterals = [...js.matchAll(/localStorage\.(?:getItem|setItem|removeItem)\(\s*["'`]([^"'`]*)["'`]/g)].map((match) => match[1]);
assert.deepEqual(storageLiterals, [], "localStorage must only ever be touched through AI_STORAGE_KEY");
assert.match(js, /localStorage\.getItem\(AI_STORAGE_KEY\)/, "the settings must be read back through the one key");
assert.match(js, /localStorage\.setItem\(AI_STORAGE_KEY,/, "the settings must be written through the one key");
assert.match(js, /localStorage\.removeItem\(AI_STORAGE_KEY\)/, "Forget settings must clear that one key");
assert.doesNotMatch(js, /sessionStorage|indexedDB|document\.cookie/, "no other browser store may be used");

assert.doesNotMatch(js, /console\s*\./, "app.js must not log anything at all, so the key can never be logged");
assert.doesNotMatch(js, /encodeURIComponent\(ai\.apiKey\)/, "the key must never be encoded into a URL");
assert.doesNotMatch(js, /[?&][a-zA-Z]*[Kk]ey=/, "the key must never travel in a query string");
assert.doesNotMatch(js, /localStorage\.setItem\(AI_STORAGE_KEY,\s*ai\.apiKey\)/, "the store must hold the settings object, not the bare key");
assert.match(js, /body: \{\s*baseUrl: ai\.baseUrl,\s*apiKey: ai\.apiKey,/, "the key only travels in a JSON request body");

/* ---------------------------------------------------------------- AI: honest states + the judge */

assert.match(js, /AI features are off — the game is fully playable without them\./, "the off state must say the game is still playable");
assert.match(js, /"Ready: " \+ ai\.model \+ " at " \+ ai\.baseUrl/, "#aiStatus must name the model and the endpoint once it is ready");
assert.match(js, /Not configured — enter a base URL, key and model, then Save\./, "the unconfigured state must say exactly what to do");
assert.match(js, /const ready = ai\.enabled && aiConfigured\(\) && !!state\.world;/, "the judge is only usable when AI is on, configured and a world is open");
assert.match(js, /\$\("aiJudgeButton"\)\.disabled = !ready;/, "the judge button must be disabled when AI is off");
assert.match(js, /function aiConfigured\(\)/, "there must be one definition of what a usable setup is");

assert.match(js, /judge: "\/api\/ai\/judge"/, "the judge must POST to /api/ai/judge");
assert.match(js, /puzzles: "\/api\/ai\/puzzles"/, "the generator must POST to /api/ai/puzzles");
assert.match(js, /question: question/, "the judge request must carry the question");
assert.match(js, /worldId: state\.worldId/, "the judge request must carry the world id");
assert.match(js, /result\.data\.verdict/, "the verdict must come from the response's verdict field");
assert.match(js, /box\.setAttribute\("data-approved", ai\.verdict\.approved \? "true" : "false"\)/, "the verdict card must carry data-approved");
assert.match(js, /ai\.verdict\.reason/, "the verdict card must show the reason");
assert.match(js, /ai\.verdict\.model/, "the verdict card must name the model that answered");
assert.match(js, /if \(result\.data && result\.data\.state\) applyState\(result\.data\.state/, "an approved verdict must render the updated state");
assert.match(js, /setJudgeError\(result\.error\)/, "a failed judge call must show the API's own message");
assert.match(js, /box\.hidden = text === "";/, "an error panel must be hidden again once it is cleared");
assert.match(js, /usedAi/, "the response's own usedAi flag must be reported, not assumed");

/* ---------------------------------------------------------------- AI: the puzzle generator */

assert.match(js, /const AI_GOAL_KINDS = \["star", "cross", "candles", "chest"\];/, "the four goal kinds must be known");
for (const kind of ["star", "cross", "candles", "chest"]) {
  assert.match(html, new RegExp(`<option value="${kind}">`), `the goal-kind select must offer ${kind}`);
}
assert.match(html, /id="aiPuzzleGoalKind"/, "the goal kind must be a select");
assert.match(js, /theme: theme, goalKind: goalKind/, "the generator must send the theme and the goal kind");
assert.match(js, /state\.aiPuzzle = \{ spec: spec, model:/, "the spec must be kept in memory");
const specUses = [...js.matchAll(/aiSpec/g)].length;
assert.equal(specUses, 1, "the spec may only be posted once — as the world to play in POST /api/worlds");
assert.match(js, /body: \{ aiSpec: spec \}/, "the generated puzzle must be opened with POST /api/worlds {aiSpec}");
assert.match(js, /badge\.hidden = !made;/, "only an AI-made puzzle gets the AI badge");
assert.match(js, /"AI-made · " \+ state\.aiPuzzle\.model/, "the badge must name the model");
assert.match(html, /id="aiWorldBadge"/, "the badge must live in the world view");
assert.match(js, /state\.aiPuzzle = null;   \/\/ a built-in puzzle is never labelled as AI-made/, "picking a built-in puzzle must clear the badge");
assert.match(js, /spec \|\| typeof spec !== "object"/, "a reply without a spec must be reported, not assumed");
assert.match(js, /setPuzzleAiError\(result\.error\)/, "a failed generation must show the API's own message");
assert.match(js, /setPuzzleAiError\("The model's reply did not contain a puzzle spec\."\)/, "a reply with no spec must say so");
assert.match(html, /<p id="aiSettingsNote"[^>]*>[^<]*never stored or logged on the server/, "the note must be plain about where the key goes");
assert.match(js, /window\.localStorage\.setItem\(AI_STORAGE_KEY[\s\S]{0,120}catch \(error\)/, "a browser that refuses storage must not break the page");

/* ---------------------------------------------------------------- packaging */

assert.match(nginx, /proxy_pass http:\/\/backend:8080;/, "Nginx must proxy /api to the Go service");
assert.match(nginx, /proxy_set_header Origin \$http_origin;/, "the proxy must preserve the browser Origin header");
assert.doesNotMatch(nginx, /proxy_set_header Origin "";/, "the proxy must not erase Origin and defeat the backend allowlist");
// Ports are the project's business, but a published port must never leave loopback.
const portLines = compose.split("\n").filter((line) => /^\s*-\s*"?\d+:/.test(line.trim()) || /^\s*-\s*"?\d+\.\d+\.\d+\.\d+:\d+:/.test(line.trim()));
assert.ok(portLines.length >= 2, "both services must publish a port for local play");
portLines.forEach((line) => {
  assert.match(line, /127\.0\.0\.1:\d+:\d+/, `a published port must bind to loopback only: ${line.trim()}`);
});
assert.doesNotMatch(compose, /0\.0\.0\.0:\d+:/, "nothing may bind to every interface");
assert.match(compose, /127\.0\.0\.1:\d+:80/, "the frontend port must bind to loopback only");
assert.match(compose, /127\.0\.0\.1:\d+:8080/, "the backend port must bind to loopback only");

assert.match(dockerfile, /COPY index\.html styles\.css app\.js/, "the image must ship the static frontend");
assert.doesNotMatch(dockerfile, /node_modules|package\.json/, "the test harness must not be baked into the image");
assert.equal(pkg.scripts.test, "node frontend_test.js", "npm test must run this script");
assert.equal(pkg.scripts["test:e2e"], "playwright test e2e.spec.js", "npm run test:e2e must run the Playwright spec");
assert.equal(pkg.devDependencies["@playwright/test"], "1.63.0", "Playwright must be pinned to 1.63.0");

/* ---------------------------------------------------------------- single functions, in isolation */

// Reads one function's source out of app.js so its body can be checked on its own.
function functionBody(name) {
  const start = js.indexOf("function " + name + "(");
  assert.ok(start !== -1, `app.js should define ${name}()`);
  let depth = 0;
  const opened = js.indexOf("{", start);
  for (let index = opened; index < js.length; index += 1) {
    if (js[index] === "{") depth += 1;
    if (js[index] === "}") {
      depth -= 1;
      if (depth === 0) return js.slice(opened, index + 1);
    }
  }
  return js.slice(opened);
}

const retryBody = functionBody("showError");
assert.match(retryBody, /typeof retry === "function" \? retry : boot/, "a failure with no retry must still offer a way forward");
const enqueueBody = functionBody("enqueue");
assert.match(enqueueBody, /catch \(error\)/, "a thrown error inside a task must be caught");
assert.match(enqueueBody, /showError\(/, "a thrown error must land in the error panel");
assert.doesNotMatch(enqueueBody, /throw /, "a thrown error must never escape to the console");

const spawnBody = functionBody("spawnPhrase");
assert.match(spawnBody, /ticks: SETTLE_TICKS/, "a spawn must settle with a two-tick step");
assert.match(spawnBody, /showError\(result\.error/, "a rejected spawn must show the API's message");
assert.match(spawnBody, /renderHistory\(\)/, "a spawn must appear in the notebook history");
assert.match(spawnBody, /chosenTarget && chosenTarget\.x != null && chosenTarget\.y != null/, "the hint's own cell must be honoured by the spawn route");

const judgeBody = functionBody("askJudge");
assert.match(judgeBody, /verdict\.approved === true/, "approval must be an explicit true from the API");
assert.doesNotMatch(judgeBody, /approved: true,/, "the UI must never invent an approval");

const hintBody = functionBody("chooseHintOption");
assert.doesNotMatch(hintBody, /option\.correct|dataset\.correct/, "pressing a choice must not consult a pre-marked answer");
assert.match(hintBody, /hintChoiceFeedback\(text\)/, "pressing a choice must report its outcome honestly");
assert.match(hintBody, /"\/hint\/choose"/, "the server must place the choice so the spot is re-proved when it is pressed");
assert.match(hintBody, /data\.movedSpot/, "a choice that had to move must be reported to the player");

console.log("frontend checks passed");
console.log(`  ${referencedIds.size} element ids referenced, ${htmlIds.length} in index.html`);
console.log(`  ${Object.keys(SHAPE_ART).length} shapes evaluated from app.js source`);
