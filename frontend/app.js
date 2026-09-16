/* Scribblebox — the game frontend.
 *
 * Vanilla JavaScript: no framework, no build step, no dependency, no external asset.
 * Every object in the world is a tidy little inline-SVG icon, drawn from one shared shape map, and
 * the player is a small friendly character holding a notebook. The browser talks to the Go service
 * through same-origin /api/... routes (Nginx proxies them inside Docker), so nothing needs a host.
 * routes (Nginx proxies them inside Docker), so nothing here needs a host name.
 *
 * The whole UI is driven from one `state` object that the API hands back after every call.
 */
(function () {
  "use strict";

  /* ------------------------------------------------------------------ routes */

  const ROUTES = {
    health: "/api/health",
    puzzles: "/api/puzzles",
    worlds: "/api/worlds"
  };

  const STEP_TICKS = 4;      // "Advance time" — fire spreads, things settle
  const SETTLE_TICKS = 2;    // auto-step right after a spawn, so the new object settles visibly
  const WORDS_LIMIT = 8;     // autocomplete page size
  const MAX_EVENTS = 80;     // the log is a narration, not an archive

  // The five things the player can do, exactly as the API names them.
  const ACTIONS = ["left", "right", "jump", "take", "drop"];

  /* ------------------------------------------------------------------ AI (optional) */

  // The one and only thing this app ever stores in the browser.
  const AI_STORAGE_KEY = "scribblebox.ai";

  // The four goal kinds the puzzle generator may be asked for.
  const AI_GOAL_KINDS = ["star", "cross", "candles", "chest"];

  const AI_ROUTES = {
    judge: "/api/ai/judge",
    puzzles: "/api/ai/puzzles",
    hint: "/api/ai/hint"
  };

  const AI_OFF_STATUS = "AI features are off — the game is fully playable without them.";
  const AI_NOT_CONFIGURED = "Not configured — enter a base URL, key and model, then Save.";

  /* ------------------------------------------------------------------ state */

  const state = {
    puzzles: [],
    preview: null,        // the puzzle whose brief/hint is on show
    puzzle: null,         // the puzzle currently being played
    world: null,          // the last state object the API sent
    worldId: "",
    events: [],           // newest first, for the log
    notebook: [],         // phrases written, in order
    suggestions: [],
    suggestionIndex: -1,
    hintOpen: false,
    placement: null,      // {x, y} chosen by clicking the drawing, null = drop from the sky
    health: null,
    retry: null,          // what the Retry button should do
    busy: false,
    aiPuzzle: null,       // {spec, model} — an AI-generated puzzle kept in memory only
    facing: "right",      // which way the little character is walking
    hint: null             // {placement, options, source, model} for the four-choice panel
    ,lastHintChoice: ""    // the choice the player last pressed, until its outcome is known
    ,hintNotice: ""        // a line to show under the next set of four choices
  };

  // Bring-your-own-key AI settings. Held in memory, mirrored to localStorage under one key.
  const ai = {
    enabled: false,
    baseUrl: "",
    apiKey: "",
    model: "",
    verdict: null         // {approved, reason, confidence, model} for the judge card
  };

  let queue = Promise.resolve(); // every API call runs in order, so state can never go backwards
  let wordTimer = 0;       // autocomplete debounce

  const $ = (id) => document.getElementById(id);
  const clamp = (value, low, high) => Math.max(low, Math.min(high, value));
  const num = (value, fallback) => {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : fallback;
  };

  /* ------------------------------------------------------------------ transport */

  // One place that talks HTTP, and one place that tells network / parse / HTTP errors apart.
  async function request(path, options) {
    const opts = Object.assign({ method: "GET", headers: {} }, options || {});
    if (opts.body !== undefined) {
      opts.headers = Object.assign({ "Content-Type": "application/json" }, opts.headers);
      opts.body = JSON.stringify(opts.body);
    }
    let response;
    try {
      response = await fetch(path, opts);
    } catch (error) {
      return { ok: false, network: true, error: "Cannot reach the Scribblebox service. Make sure the Go API is running (docker compose up)." };
    }

    let raw = "";
    try {
      raw = await response.text();
    } catch (error) {
      raw = "";
    }

    let data = null;
    let parsed = false;
    if (raw) {
      try {
        data = JSON.parse(raw);
        parsed = true;
      } catch (error) {
        data = null;
      }
    }

    if (!response.ok) {
      const detail = data && typeof data.error === "string" ? data.error : "";
      return {
        ok: false,
        status: response.status,
        data: data,
        error: detail || ("The service answered HTTP " + response.status + (raw ? " — " + raw.slice(0, 160) : ""))
      };
    }
    if (raw && !parsed) {
      return { ok: false, status: response.status, error: "The service sent a reply the notebook could not read." };
    }
    return { ok: true, status: response.status, data: data };
  }

  // Runs one task at a time, and turns any thrown error into the error panel instead of a crash.
  function enqueue(task, retry) {
    const run = async () => {
      state.busy = true;
      try {
        await task();
      } catch (error) {
        const message = error && error.message ? error.message : String(error);
        showError("Something in the page went wrong: " + message, retry || boot);
      } finally {
        state.busy = false;
        $("worldStatus").setAttribute("aria-busy", "false");
      }
    };
    queue = queue.then(run, run);
    return queue;
  }

  /* ------------------------------------------------------------------ errors + status */

  function showError(message, retry) {
    state.retry = typeof retry === "function" ? retry : boot;
    $("errorMessage").textContent = String(message || "The last request failed.");
    $("errorPanel").hidden = false;
  }

  function hideError() {
    $("errorPanel").hidden = true;
    state.retry = null;
  }

  function setPuzzlesStatus(text) {
    $("puzzlesStatus").textContent = text;
  }

  /* ------------------------------------------------------------------ boot */

  async function boot() {
    setPuzzlesStatus("Looking for puzzles…");
    await enqueue(async () => {
      const health = await request(ROUTES.health);
      if (health.ok && health.data) {
        state.health = health.data;
        const words = num(health.data.words, 0);
        const puzzles = num(health.data.puzzles, 0);
        $("dictionaryNote").textContent = "The dictionary holds " + words + " words and " + puzzles + " puzzles.";
      } else {
        state.health = null;
        $("dictionaryNote").textContent = "The dictionary is not answering yet.";
        showError(health.error, boot);
      }

      const puzzles = await request(ROUTES.puzzles);
      if (!puzzles.ok) {
        setPuzzlesStatus("The puzzle list could not be loaded.");
        showError(puzzles.error, boot);
        return;
      }
      const list = puzzles.data && Array.isArray(puzzles.data.puzzles) ? puzzles.data.puzzles : [];
      state.puzzles = list;
      if (!state.preview) state.preview = list[0] || null;
      renderPuzzleList();
      if (!list.length) {
        setPuzzlesStatus("The service knows no puzzles yet.");
        return;
      }
      setPuzzlesStatus(list.length === 1 ? "1 puzzle ready." : list.length + " puzzles ready.");
      if (health.ok) hideError();
    }, boot);
  }

  /* ------------------------------------------------------------------ puzzles */

  function renderPuzzleList() {
    const list = $("puzzleList");
    list.innerHTML = "";
    state.puzzles.forEach((puzzle) => {
      const item = document.createElement("li");
      const button = document.createElement("button");
      button.type = "button";
      button.className = "puzzle-card";
      button.dataset.puzzle = String(puzzle.id || "");
      button.setAttribute("aria-label", "Play " + (puzzle.title || puzzle.id || "this puzzle"));
      if (state.puzzle && puzzle.id === state.puzzle.id) button.setAttribute("aria-current", "true");

      const title = document.createElement("span");
      title.className = "puzzle-title";
      title.textContent = String(puzzle.title || puzzle.id || "Untitled puzzle");

      const brief = document.createElement("span");
      brief.className = "puzzle-brief";
      brief.textContent = String(puzzle.brief || "");
      brief.title = String(puzzle.brief || "");

      button.appendChild(title);
      button.appendChild(brief);
      button.addEventListener("click", () => {
        if (state.puzzle && puzzle.id === state.puzzle.id && state.world) {
          showBrief(puzzle);
          $("worldSvg").focus();
          return;
        }
        openPuzzle(puzzle);
      });
      item.appendChild(button);
      list.appendChild(item);
    });
  }

  function showBrief(puzzle) {
    if (!puzzle) {
      $("puzzleBrief").textContent = "Pick a puzzle and its brief appears here.";
      $("puzzleHint").textContent = "";
      $("puzzleHint").hidden = true;
      return;
    }
    $("puzzleBrief").textContent = String(puzzle.brief || "No brief was given for this puzzle.");
    $("puzzleHint").textContent = String(puzzle.hint || "No hint for this puzzle — use your head.");
    $("puzzleHint").hidden = !state.hintOpen;
    $("puzzleHintButton").textContent = state.hintOpen ? "Hide hint" : "Show hint";
    $("puzzleHintButton").setAttribute("aria-expanded", state.hintOpen ? "true" : "false");
  }

  function toggleHint() {
    state.hintOpen = !state.hintOpen;
    showBrief(state.preview);
  }

  function openPuzzle(puzzle) {
    if (!puzzle) return;
    state.preview = puzzle;
    showBrief(puzzle);
    renderPuzzleList();
    setPuzzlesStatus("Opening “" + (puzzle.title || puzzle.id) + "”…");
    enqueue(async () => {
      const result = await request(ROUTES.worlds, { method: "POST", body: { puzzleId: puzzle.id } });
      if (!result.ok) {
        setPuzzlesStatus("That puzzle could not be opened.");
        showError(result.error, () => openPuzzle(puzzle));
        return;
      }
      state.puzzle = puzzle;
      state.placement = null;
      state.notebook = [];
      state.events = [];
      state.suggestions = [];
      state.suggestionIndex = -1;
      state.aiPuzzle = null;   // a built-in puzzle is never labelled as AI-made
      ai.verdict = null;
      state.facing = "right";
      setJudgeError("");
      setHintError("");
      renderHint(null, null);
      renderSuggestions();
      hideError();
      applyState(result.data, result.data && result.data.events);
      setPuzzlesStatus("Playing “" + (puzzle.title || puzzle.id) + "”.");
      setSpawnNote("Write something — it lands " + describePlacement() + ".", "quiet");
      $("worldSvg").focus();
    }, () => openPuzzle(puzzle));
  }

  /* ------------------------------------------------------------------ state in */

  function ingestEvents(world, delta) {
    const incoming = Array.isArray(delta) ? delta : [];
    if (incoming.length) state.events = incoming.slice().reverse().concat(state.events);
    const fromState = world && Array.isArray(world.events) ? world.events : [];
    fromState.slice().reverse().forEach((event) => {
      if (!state.events.includes(event)) state.events.push(event);
    });
    state.events = state.events.slice(0, MAX_EVENTS);
  }

  function applyState(payload, delta) {
    const world = payload && payload.state ? payload.state : payload;
    if (!world || typeof world !== "object" || !Array.isArray(world.terrain)) return false;
    state.world = world;
    state.worldId = String(world.id || state.worldId || "");
    ingestEvents(world, delta);
    render();
    return true;
  }

  function render() {
    const world = state.world;
    if (!world) return;
    renderWorldSvg(world);
    renderStatus(world);
    renderGoal(world);
    renderEvents();
    renderEntities(world);
    renderHistory();
    renderPlacement(world);
    renderControls(world);
    renderAiBadge();
    renderJudge();
    renderHintAi();
    refreshHintFeedback();
      }

  /* ------------------------------------------------------------------ rendering: text */

  function renderStatus(world) {
    const puzzle = world.puzzle || state.puzzle || {};
    const player = world.player || { x: 0, y: 0 };
    const entities = Array.isArray(world.entities) ? world.entities : [];
    const holding = entities.filter((entity) => num(entity.id, -1) === num(player.holding, 0) && num(player.holding, 0) !== 0);
    const heldName = holding.length && holding[0].object ? String(holding[0].object.name || holding[0].object.key || "something") : "";

    $("worldName").textContent = String(puzzle.title || ("World " + String(world.id || "").slice(0, 8)));
    $("tickCount").textContent = "tick " + num(world.ticks, 0);
    $("playerStatus").textContent = "you are at x " + num(player.x, 0) + ", y " + num(player.y, 0) +
      (heldName ? " · holding " + heldName : "") + (player.onGround === false ? " · in the air" : "");
    $("objectStatus").textContent = countText(entities.length);
  }

  function countText(count) {
    return count + (count === 1 ? " object" : " objects");
  }

  function renderGoal(world) {
    const goal = world.goal || {};
    const puzzle = world.puzzle || state.puzzle || {};
    $("goalTitle").textContent = String(goal.title || puzzle.brief || "Reach the goal.");
    $("goalProgress").textContent = String(goal.progress || (goal.met ? "the goal is met" : "not met yet"));
    const solved = world.solved === true || goal.met === true;
    $("solvedBanner").hidden = !solved;
    $("goalBanner").classList.toggle("is-solved", solved);
    $("solvedGoal").textContent = solved ? String(goal.title || puzzle.brief || "Goal met") : "";
  }

  function renderEvents() {
    const list = $("eventLog");
    list.innerHTML = "";
    if (!state.events.length) {
      const item = document.createElement("li");
      item.className = "entity-empty";
      item.textContent = "Nothing has happened yet.";
      list.appendChild(item);
    } else {
      state.events.forEach((event, index) => {
        const item = document.createElement("li");
        item.textContent = String(event);
        if (index === 0) item.classList.add("is-new");
        list.appendChild(item);
      });
    }
    $("eventCount").textContent = state.events.length
      ? state.events.length + " events, newest first"
      : "nothing yet";
  }

  function renderEntities(world) {
    const list = $("entityList");
    const entities = Array.isArray(world.entities) ? world.entities : [];
    $("entityCount").textContent = countText(entities.length);
    list.innerHTML = "";

    if (!entities.length) {
      const empty = document.createElement("li");
      empty.className = "entity-empty";
      empty.textContent = "Nothing is in the world yet — write an object in the notebook.";
      list.appendChild(empty);
      return;
    }

    entities.forEach((entity) => {
      const object = entity && entity.object ? entity.object : {};
      const item = document.createElement("li");
      item.className = "entity";
      item.dataset.key = String(object.key || "");
      item.dataset.entityId = String(num(entity.id, 0));
      item.dataset.burning = entity.burning ? "true" : "false";
      item.dataset.opened = entity.opened ? "true" : "false";
      item.dataset.cut = entity.cut ? "true" : "false";
      item.dataset.powered = entity.powered ? "true" : "false";
      item.dataset.held = entity.held ? "true" : "false";
      item.dataset.at = num(entity.x, 0) + "," + num(entity.y, 0);

      const head = document.createElement("div");
      head.className = "entity-head";
      const name = document.createElement("span");
      name.className = "entity-name";
      name.textContent = String(object.name || object.noun || object.key || "something");
      const key = document.createElement("span");
      key.className = "entity-key";
      key.textContent = String(object.key || "unknown");
      head.appendChild(name);
      head.appendChild(key);

      const where = document.createElement("div");
      where.className = "entity-where";
      where.textContent = "at x " + num(entity.x, 0) + ", y " + num(entity.y, 0) +
        " · " + Math.max(1, num(entity.w, 1)) + "×" + Math.max(1, num(entity.h, 1)) +
        " · " + String(object.category || "thing") +
        (object.approximate ? " · improvised" : "");

      item.appendChild(head);
      item.appendChild(where);

      const tags = document.createElement("div");
      tags.className = "entity-tags";
      const flags = [];
      if (entity.burning) flags.push({ text: "burning", kind: "flag flag-alarm" });
      if (entity.opened) flags.push({ text: "opened", kind: "flag" });
      if (entity.cut) flags.push({ text: "cut", kind: "flag" });
      if (entity.powered) flags.push({ text: "powered", kind: "flag" });
      if (entity.held) flags.push({ text: "in hand", kind: "flag" });
      if (!flags.length) flags.push({ text: "intact", kind: "flag" });
      flags.forEach((flag) => {
        const span = document.createElement("span");
        span.className = "tag " + flag.kind;
        span.textContent = flag.text;
        tags.appendChild(span);
      });

      const tagList = Array.isArray(object.tags) ? object.tags : [];
      tagList.forEach((tag) => {
        const span = document.createElement("span");
        span.className = "tag";
        span.textContent = String(tag);
        tags.appendChild(span);
      });
      item.appendChild(tags);
      list.appendChild(item);
    });
  }

  function renderHistory() {
    const list = $("notebookHistory");
    list.innerHTML = "";
    if (!state.notebook.length) {
      const item = document.createElement("li");
      item.className = "history-empty";
      item.textContent = "nothing written yet";
      list.appendChild(item);
      return;
    }
    state.notebook.forEach((entry) => {
      const item = document.createElement("li");
      item.textContent = entry.phrase;
      const key = document.createElement("span");
      key.className = "history-key";
      key.textContent = " → " + entry.key + (entry.approximate ? " ?" : "");
      item.appendChild(key);
      list.appendChild(item);
    });
  }

  function renderControls(world) {
    const open = !!world;
    ["btnLeft", "btnRight", "btnJump", "btnTake", "btnDrop", "btnStep", "btnReset", "btnDropFromSky", "hintButton"].forEach((id) => {
      $(id).disabled = !open;
    });
    $("spawnButton").disabled = !open || $("notebookInput").value.trim() === "";
  }

  function renderPlacement(world) {
    if (!world) {
      $("placementHint").textContent = "Pick a puzzle to open a world.";
      return;
    }
    const target = placementCell(world);
    const source = target.source === "chosen"
      ? "on the marked cell (x " + target.x + ", y " + target.y + ")"
      : "from the sky above you (x " + target.x + ", y " + target.y + ")";
    $("placementHint").textContent = "The next object will land " + source +
      ". " + (target.source === "chosen"
        ? "Press “Drop from the sky” to go back to the default spot."
        : "Click a cell in the drawing to choose somewhere else.");
  }

  function describePlacement() {
    if (!state.world) return "in the world";
    const target = placementCell(state.world);
    return target.source === "chosen" ? "on the marked cell" : "from the sky above you";
  }

  function setSpawnNote(text, kind) {
    const note = $("spawnNote");
    note.textContent = String(text || "");
    note.className = "spawn-note" + (kind ? " is-" + kind : "");
  }

  /* ------------------------------------------------------------------ placement */

  function placementCell(world) {
    const width = Math.max(1, num(world.width, 1));
    const height = Math.max(1, num(world.height, 1));
    if (state.placement) {
      return {
        x: clamp(num(state.placement.x, 0), 0, width - 1),
        y: clamp(num(state.placement.y, 0), 0, height - 1),
        source: "chosen"
      };
    }
    const player = world.player || { x: 0, y: 0 };
    return {
      x: clamp(num(player.x, 0), 0, width - 1),
      y: clamp(num(player.y, 0) - 3, 0, height - 1),
      source: "sky"
    };
  }

  /* ------------------------------------------------------------------ notebook */

  function onNotebookInput() {
    const value = $("notebookInput").value;
    $("spawnButton").disabled = !state.world || value.trim() === "";
    window.clearTimeout(wordTimer);
    state.suggestionIndex = -1;
    const query = value.trim();
    if (!query) {
      state.suggestions = [];
      renderSuggestions();
      return;
    }
    wordTimer = window.setTimeout(() => {
      const asked = query;
      enqueue(async () => {
        const result = await request("/api/words?q=" + encodeURIComponent(asked) + "&limit=" + WORDS_LIMIT);
        // The list is a convenience: a failed lookup must never take the game down.
        if (!result.ok) {
          state.suggestions = [];
          state.suggestionIndex = -1;
          renderSuggestions();
          return;
        }
        const words = result.data && Array.isArray(result.data.words) ? result.data.words : [];
        if ($("notebookInput").value.trim() !== asked) return;
        state.suggestions = words.slice(0, WORDS_LIMIT);
        state.suggestionIndex = -1;
        renderSuggestions();
        const dictionary = result.data && num(result.data.dictionary, 0);
        if (dictionary) $("dictionaryNote").textContent = "The dictionary holds " + dictionary + " words.";
      }, boot);
    }, 140);
  }

  function renderSuggestions() {
    const list = $("notebookSuggestions");
    const input = $("notebookInput");
    list.innerHTML = "";
    if (!state.suggestions.length) {
      list.hidden = true;
      input.setAttribute("aria-expanded", "false");
      input.removeAttribute("aria-activedescendant");
      return;
    }
    list.hidden = false;
    input.setAttribute("aria-expanded", "true");
    state.suggestions.forEach((word, index) => {
      const item = document.createElement("li");
      item.className = "suggestion" + (index === state.suggestionIndex ? " is-active" : "");
      item.setAttribute("role", "option");
      item.setAttribute("aria-selected", index === state.suggestionIndex ? "true" : "false");
      item.id = "notebook-suggestion-" + index;
      item.dataset.word = word;
      item.textContent = String(word);
      item.addEventListener("mousedown", (event) => {
        event.preventDefault();
        chooseSuggestion(index, true);
      });
      list.appendChild(item);
    });
    if (state.suggestionIndex >= 0) input.setAttribute("aria-activedescendant", "notebook-suggestion-" + state.suggestionIndex);
    else input.removeAttribute("aria-activedescendant");
  }

  function moveSuggestion(step) {
    if (!state.suggestions.length) return;
    const count = state.suggestions.length;
    const next = state.suggestionIndex + step;
    state.suggestionIndex = ((next % count) + count) % count;
    renderSuggestions();
  }

  function chooseSuggestion(index, shouldSpawn) {
    const word = state.suggestions[index];
    if (!word) return;
    $("notebookInput").value = String(word);
    state.suggestions = [];
    state.suggestionIndex = -1;
    renderSuggestions();
    $("spawnButton").disabled = !state.world || String(word).trim() === "";
    $("notebookInput").focus();
    if (shouldSpawn) spawnPhrase(String(word));
  }

  function onSubmitNotebook(event) {
    event.preventDefault();
    spawnPhrase($("notebookInput").value);
  }

  /* ------------------------------------------------------------------ actions */

  function spawnPhrase(phrase, chosenTarget, fromHint) {
    const text = String(phrase || "").trim();
    if (!text) return;
    if (!state.world) {
      showError("There is no world open yet — pick a puzzle first.", boot);
      return;
    }
    const width = Math.max(1, num(state.world.width, 1));
    const height = Math.max(1, num(state.world.height, 1));
    const target = chosenTarget && chosenTarget.x != null && chosenTarget.y != null
      ? { x: clamp(num(chosenTarget.x, 0), 0, width - 1), y: clamp(num(chosenTarget.y, 0), 0, height - 1), source: "chosen" }
      : placementCell(state.world);
    enqueue(async () => {
      setSpawnNote("Writing “" + text + "”…", "quiet");
      $("worldSvg").setAttribute("aria-busy", "true");
      const result = await request(
        ROUTES.worlds + "/" + encodeURIComponent(state.worldId) + "/spawn",
        { method: "POST", body: { phrase: text, x: target.x, y: target.y } }
      );
      $("worldSvg").setAttribute("aria-busy", "false");

      if (!result.ok) {
        setSpawnNote("That did not work: " + result.error, "error");
        showError(result.error, () => spawnPhrase(text, chosenTarget, fromHint));
        if (fromHint) {
          // The hint spot fills up as the player tries choices, so a choice that
          // cannot be placed there must not leave four options that no longer fit:
          // say so and fetch four fresh ones for the world as it stands.
          state.hintNotice = "“" + text + "” could not be placed at the hint spot (" + result.error +
            "). Here are four fresh choices.";
          state.lastHintChoice = "";
          askForHint();
        }
        return;
      }
      hideError();

      const spawned = result.data && result.data.spawned ? result.data.spawned : null;
      const approximate = !!(result.data && result.data.approximate) || !!(spawned && spawned.approximate);
      applyState(result.data, result.data && result.data.events);

      state.notebook = state.notebook.concat([{
        phrase: text,
        key: String((spawned && spawned.key) || text),
        approximate: approximate,
        x: target.x,
        y: target.y
      }]);
      renderHistory();

      const note = (result.data && result.data.note) || (spawned && spawned.note) || "";
      const where = "at x " + target.x + ", y " + target.y;
      const verdict = approximate
        ? " The book has never heard of that, so it improvised."
        : " The book knew that word.";
      setSpawnNote((note ? note + " " : "") + "“" + text + "” is in the world " + where + "." + verdict,
        approximate ? "approximate" : "exact");

      $("notebookInput").value = "";
      $("spawnButton").disabled = true;
      $("notebookInput").focus();

      // One short step so the new object settles (falls, spreads, floats) before the player looks away.
      const settled = await request(
        ROUTES.worlds + "/" + encodeURIComponent(state.worldId) + "/step",
        { method: "POST", body: { ticks: SETTLE_TICKS } }
      );
      if (settled.ok) applyState(settled.data, settled.data && settled.data.events);
      if (fromHint) hintChoiceFeedback(text);
    }, () => spawnPhrase(text, chosenTarget, fromHint));
  }

  function doAction(action) {
    if (ACTIONS.indexOf(action) === -1) return;
    if (!state.world) return;
    if (action === "left") state.facing = "left";
    if (action === "right") state.facing = "right";
    enqueue(async () => {
      const result = await request(
        ROUTES.worlds + "/" + encodeURIComponent(state.worldId) + "/act",
        { method: "POST", body: { action: action } }
      );
      if (!result.ok) {
        showError(result.error, () => doAction(action));
        return;
      }
      hideError();
      applyState(result.data, result.data && result.data.events);
    }, () => doAction(action));
  }

  function advanceTime() {
    if (!state.world) return;
    enqueue(async () => {
      const result = await request(
        ROUTES.worlds + "/" + encodeURIComponent(state.worldId) + "/step",
        { method: "POST", body: { ticks: STEP_TICKS } }
      );
      if (!result.ok) {
        showError(result.error, advanceTime);
        return;
      }
      hideError();
      applyState(result.data, result.data && result.data.events);
    }, advanceTime);
  }

  function resetPuzzle() {
    if (!state.world) return;
    enqueue(async () => {
      const result = await request(
        ROUTES.worlds + "/" + encodeURIComponent(state.worldId) + "/reset",
        { method: "POST", body: {} }
      );
      if (!result.ok) {
        showError(result.error, resetPuzzle);
        return;
      }
      hideError();
      state.placement = null;
      state.notebook = [];
      state.events = [];
      ai.verdict = null;
      state.facing = "right";
      setHintError("");
      renderHint(null, null);
      applyState(result.data, result.data && result.data.events);
      setSpawnNote("The puzzle is back at the start.", "quiet");
      renderJudge();
    }, resetPuzzle);
  }

  function dropFromSky() {
    state.placement = null;
    if (state.world) {
      renderPlacement(state.world);
      renderWorldSvg(state.world);
    }
    $("notebookInput").focus();
  }

  function chooseAnotherPuzzle() {
    renderPuzzleList();
    setPuzzlesStatus(state.puzzles.length ? "Pick any puzzle." : "No puzzles loaded yet.");
    const first = $("puzzleList").querySelector("button");
    if (first) first.focus();
    window.scrollTo({ top: 0, behavior: "smooth" });
  }

  /* ------------------------------------------------------------------ AI (optional) */

  // Settings live under one localStorage key and nowhere else. The key never leaves this browser
  // except as part of a request the player explicitly asked for, and it is never logged.
  function loadAiSettings() {
    let stored = null;
    try {
      const raw = window.localStorage.getItem(AI_STORAGE_KEY);
      if (raw) stored = JSON.parse(raw);
    } catch (error) {
      stored = null; // no storage, or unreadable: the game simply starts with AI off
    }
    if (stored && typeof stored === "object") {
      ai.enabled = stored.enabled === true;
      ai.baseUrl = typeof stored.baseUrl === "string" ? stored.baseUrl : "";
      ai.apiKey = typeof stored.apiKey === "string" ? stored.apiKey : "";
      ai.model = typeof stored.model === "string" ? stored.model : "";
    }
    $("aiBaseUrl").value = ai.baseUrl;
    $("aiApiKey").value = ai.apiKey;
    $("aiApiKey").type = "password";
    $("aiModel").value = ai.model;
    $("aiToggle").checked = ai.enabled;
    renderAi();
  }

  function persistAi() {
    const payload = { enabled: ai.enabled, baseUrl: ai.baseUrl, apiKey: ai.apiKey, model: ai.model };
    try {
      window.localStorage.setItem(AI_STORAGE_KEY, JSON.stringify(payload));
    } catch (error) {
      // A browser that refuses storage still gets a fully working session: memory keeps the settings.
    }
  }

  function aiConfigured() {
    return !!(ai.baseUrl && ai.apiKey && ai.model);
  }

  function renderAi() {
    $("aiToggle").checked = ai.enabled;
    $("aiStatus").textContent = !ai.enabled
      ? AI_OFF_STATUS
      : (aiConfigured() ? "Ready: " + ai.model + " at " + ai.baseUrl : AI_NOT_CONFIGURED);
    renderJudge();
    renderHintAi();
    renderPuzzleAi();
  }

  function saveAiSettings(event) {
    if (event) event.preventDefault();
    const baseUrl = $("aiBaseUrl").value.trim();
    const apiKey = $("aiApiKey").value.trim();
    const model = $("aiModel").value.trim();
    const enabled = $("aiToggle").checked;

    if (baseUrl && !/^[a-zA-Z][a-zA-Z0-9+.-]*:\/\//.test(baseUrl)) {
      $("aiStatus").textContent = "That base URL has no scheme — it should start with http:// or https://.";
      return;
    }
    if (enabled && !(baseUrl && apiKey && model)) {
      $("aiStatus").textContent = AI_NOT_CONFIGURED;
      ai.enabled = enabled;
      persistAi();
      renderJudge();
      renderPuzzleAi();
      return;
    }

    ai.baseUrl = baseUrl;
    ai.apiKey = apiKey;
    ai.model = model;
    ai.enabled = enabled;
    persistAi();
    renderAi();
  }

  function toggleKeyReveal() {
    const field = $("aiApiKey");
    const shown = field.type === "text";
    field.type = shown ? "password" : "text";
    $("aiKeyReveal").setAttribute("aria-pressed", shown ? "false" : "true");
    $("aiKeyReveal").textContent = shown ? "Show key" : "Hide key";
    field.focus();
  }

  function forgetAi() {
    ai.enabled = false;
    ai.baseUrl = "";
    ai.apiKey = "";
    ai.model = "";
    ai.verdict = null;
    state.aiPuzzle = null;
    try {
      window.localStorage.removeItem(AI_STORAGE_KEY);
    } catch (error) {
      // nothing stored, nothing to remove
    }
    $("aiBaseUrl").value = "";
    $("aiApiKey").value = "";
    $("aiApiKey").type = "password";
    $("aiModel").value = "";
    $("aiKeyReveal").textContent = "Show key";
    $("aiKeyReveal").setAttribute("aria-pressed", "false");
    setJudgeError("");
    setPuzzleAiError("");
    $("aiPuzzleStatus").textContent = "AI is off, so nothing is invented. Turn AI on above to use this.";
    renderAi();
    renderAiBadge();
  }

  function renderJudge() {
    const box = $("aiVerdict");
    const ready = ai.enabled && aiConfigured() && !!state.world;
    $("aiJudgeButton").disabled = !ready;

    if (!ai.enabled) {
      box.className = "ai-verdict";
      box.removeAttribute("data-approved");
      box.textContent = AI_OFF_STATUS;
      return;
    }
    if (!aiConfigured()) {
      box.className = "ai-verdict";
      box.removeAttribute("data-approved");
      box.textContent = AI_NOT_CONFIGURED;
      return;
    }
    if (ai.verdict) {
      box.className = "ai-verdict " + (ai.verdict.approved ? "is-approved" : "is-refused");
      box.setAttribute("data-approved", ai.verdict.approved ? "true" : "false");
      box.textContent = (ai.verdict.approved ? "The judge approves: " : "The judge is not convinced: ") +
        ai.verdict.reason +
        (ai.verdict.confidence ? " (confidence: " + ai.verdict.confidence + ")" : "") +
        (ai.verdict.model ? " — answered by " + ai.verdict.model : "") +
        (ai.verdict.usedAi === false ? " (the server did not use the AI for this answer)" : "");
      return;
    }
    box.className = "ai-verdict";
    box.removeAttribute("data-approved");
    box.textContent = state.world
      ? "Ask the judge whether an idea would work. A yes only says the idea sounds sound — the world still has to show it."
      : "Open a puzzle first, then ask the judge about your idea.";
  }

  function setJudgeError(message) {
    const box = $("aiJudgeError");
    const text = String(message || "").trim();
    box.textContent = text;
    box.hidden = text === "";
  }

  function askJudge() {
    const question = $("aiJudgeQuestion").value.trim();
    if (!question) {
      setJudgeError("Type the idea you want judged first.");
      return;
    }
    if (!ai.enabled) {
      setJudgeError(AI_OFF_STATUS);
      return;
    }
    if (!aiConfigured()) {
      setJudgeError(AI_NOT_CONFIGURED);
      return;
    }
    if (!state.world) {
      setJudgeError("There is no world open yet — pick a puzzle first.");
      return;
    }

    enqueue(async () => {
      setJudgeError("");
      ai.verdict = null;
      $("aiVerdict").className = "ai-verdict";
      $("aiVerdict").removeAttribute("data-approved");
      $("aiVerdict").textContent = "Asking " + ai.model + "…";
      $("aiJudgeButton").disabled = true;

      const result = await request(AI_ROUTES.judge, {
        method: "POST",
        body: {
          baseUrl: ai.baseUrl,
          apiKey: ai.apiKey,
          model: ai.model,
          worldId: state.worldId,
          question: question
        }
      });

      if (!result.ok) {
        // The API's own message goes on screen word for word: a refused key must look refused.
        $("aiVerdict").textContent = "No verdict came back.";
        setJudgeError(result.error);
        renderJudge();
        return;
      }

      const verdict = result.data && result.data.verdict ? result.data.verdict : null;
      if (!verdict) {
        $("aiVerdict").textContent = "No verdict came back.";
        setJudgeError("The judge answered without a verdict.");
        renderJudge();
        return;
      }

      ai.verdict = {
        approved: verdict.approved === true,
        reason: String(verdict.reason || "no reason given"),
        confidence: String(verdict.confidence || ""),
        model: String((result.data && result.data.model) || ai.model),
        // The API says whether the answer really came from the AI; if it did not, the card says so.
        usedAi: !(result.data && result.data.usedAi === false)
      };
      if (result.data && result.data.state) applyState(result.data.state, result.data.state.events);
      renderJudge();
      if (ai.verdict.approved) $("aiJudgeQuestion").value = "";
      $("aiJudgeQuestion").focus();
    }, askJudge);
  }

  function renderAiBadge() {
    const badge = $("aiWorldBadge");
    const made = !!(state.aiPuzzle && state.aiPuzzle.model);
    badge.hidden = !made;
    badge.textContent = made ? "AI-made · " + state.aiPuzzle.model : "";
    if (made) {
      badge.title = "Invented by " + state.aiPuzzle.model + " from the theme “" +
        String((state.aiPuzzle.spec && state.aiPuzzle.spec.title) || "") + "”";
    } else {
      badge.removeAttribute("title");
    }
    $("worldSvg").dataset.ai = made ? "true" : "false";
  }

  function renderPuzzleAi() {
    $("aiPuzzleButton").disabled = !(ai.enabled && aiConfigured());
    if (!ai.enabled) {
      $("aiPuzzleStatus").textContent = "AI is off, so nothing is invented. Turn AI on above to use this.";
      return;
    }
    if (!aiConfigured()) {
      $("aiPuzzleStatus").textContent = AI_NOT_CONFIGURED;
      return;
    }
    if (!state.aiPuzzle) {
      $("aiPuzzleStatus").textContent = "Give a theme and a goal kind, then invent a puzzle. It plays like any other puzzle.";
    }
  }

  function setPuzzleAiError(message) {
    const box = $("aiPuzzleError");
    const text = String(message || "").trim();
    box.textContent = text;
    box.hidden = text === "";
  }

  function generateAiPuzzle() {
    if (!ai.enabled || !aiConfigured()) {
      setPuzzleAiError(AI_NOT_CONFIGURED);
      $("aiPuzzleStatus").textContent = ai.enabled ? AI_NOT_CONFIGURED : "AI is off, so nothing is invented. Turn AI on above to use this.";
      return;
    }
    const theme = $("aiPuzzleTheme").value.trim() || "anything you like";
    const chosen = $("aiPuzzleGoalKind").value;
    const goalKind = AI_GOAL_KINDS.indexOf(chosen) >= 0 ? chosen : "star";

    enqueue(async () => {
      setPuzzleAiError("");
      $("aiPuzzleButton").disabled = true;
      $("aiPuzzleStatus").textContent = "Asking " + ai.model + " for a puzzle about " + theme + "…";

      const result = await request(AI_ROUTES.puzzles, {
        method: "POST",
        body: { baseUrl: ai.baseUrl, apiKey: ai.apiKey, model: ai.model, theme: theme, goalKind: goalKind }
      });
      $("aiPuzzleButton").disabled = false;

      if (!result.ok) {
        $("aiPuzzleStatus").textContent = "No puzzle was invented.";
        setPuzzleAiError(result.error);
        return;
      }
      const spec = result.data && result.data.spec ? result.data.spec : null;
      if (!spec || typeof spec !== "object") {
        $("aiPuzzleStatus").textContent = "No puzzle was invented.";
        setPuzzleAiError("The model's reply did not contain a puzzle spec.");
        return;
      }

      // The spec stays right here in memory — it is only ever sent back as the world to play.
      state.aiPuzzle = { spec: spec, model: String((result.data && result.data.model) || ai.model) };
      $("aiPuzzleStatus").textContent = "The puzzle came back — opening it in the world view…";
      await openAiPuzzle();
    }, generateAiPuzzle);
  }

  async function openAiPuzzle() {
    const held = state.aiPuzzle;
    if (!held || !held.spec) return;
    const spec = held.spec;

    const result = await request(ROUTES.worlds, { method: "POST", body: { aiSpec: spec } });
    if (!result.ok) {
      $("aiPuzzleStatus").textContent = "The invented puzzle could not be opened.";
      setPuzzleAiError(result.error);
      return;
    }

    state.puzzle = {
      id: spec.id,
      title: spec.title,
      brief: spec.brief,
      hint: spec.hint,
      goalKind: spec.goalKind,
      width: spec.width,
      height: spec.height
    };
    state.preview = state.puzzle;
    state.placement = null;
    state.notebook = [];
    state.events = [];
    ai.verdict = null;
    state.facing = "right";
    setJudgeError("");
    setHintError("");
    renderHint(null, null);
    showBrief(state.puzzle);
    hideError();
    applyState(result.data, result.data && result.data.events);
    renderPuzzleList();
    setPuzzlesStatus("Playing the AI-made puzzle “" + (spec.title || spec.id) + "”.");
    $("aiPuzzleStatus").textContent = "Open in the world view. The AI badge beside the title marks it as invented.";
    $("worldSvg").focus();
  }

  /* ------------------------------------------------------------------ keyboard */

  function onKeyDown(event) {
    const target = event.target;
    const inField = target && (target.tagName === "INPUT" || target.tagName === "TEXTAREA" || target.isContentEditable);

    if (event.key === "Escape") {
      if (state.suggestions.length) {
        state.suggestions = [];
        state.suggestionIndex = -1;
        renderSuggestions();
      }
      return;
    }

    if (inField) {
      // The notebook owns the arrow keys and Enter while the player is typing.
      if (target.id === "notebookInput") {
        if (event.key === "ArrowDown") {
          event.preventDefault();
          moveSuggestion(1);
          return;
        }
        if (event.key === "ArrowUp") {
          event.preventDefault();
          moveSuggestion(-1);
          return;
        }
        if (event.key === "Enter" && state.suggestionIndex >= 0) {
          event.preventDefault();
          chooseSuggestion(state.suggestionIndex, true);
          return;
        }
      }
      return;
    }

    switch (event.key) {
      case "ArrowLeft":
        event.preventDefault();
        doAction("left");
        break;
      case "ArrowRight":
        event.preventDefault();
        doAction("right");
        break;
      case " ":
      case "Spacebar":
      case "ArrowUp":
        event.preventDefault();
        doAction("jump");
        break;
      case "t":
      case "T":
        event.preventDefault();
        doAction("take");
        break;
      case "d":
      case "D":
        event.preventDefault();
        doAction("drop");
        break;
      case "s":
      case "S":
        event.preventDefault();
        advanceTime();
        break;
      default:
        break;
    }
  }

  /* ------------------------------------------------------------------ four choices (hint) */

  /* The hint offers four objects to pick from. Nothing here is told which one works — the response
   * carries no such field, so the UI cannot mark an option right or wrong before the player tries
   * it. The feedback afterwards only ever repeats what the world's own state says. */

  function hintPhraseLabel(phrase) {
    const letters = String(phrase).split(/\s+/).filter(Boolean).map((word) => word.charAt(0)).join("");
    return (letters || String(phrase)).slice(0, 3).toUpperCase();
  }

  // A small picture for the button. The words endpoint answers with the shape and
  // colour of any word the dictionary knows, so a choice shows the very icon it
  // will have in the world; an unknown word gets the blobby improvised art. Either
  // way the button is usable immediately and the drawing is never in the way.
  function hintPreviewSvg(phrase, knowledge, shape, color) {
    const known = knowledge === "known";
    const fill = color || (known ? "#c9a06a" : "#c3cad6");
    const art = known && shape && SHAPE_ART[shape]
      ? SHAPE_ART[shape](1, 1, fill)
      : (known ? SHAPE_ART.box(1, 1, fill) : SHAPE_ART.blob(1, 1, fill));
    const label = phrase + (known ? " — a word the dictionary knows" : " — not in the dictionary, so this is a stand-in drawing");
    return '<svg class="hint-preview-art" viewBox="0 0 1 1" preserveAspectRatio="xMidYMid meet" role="img" ' +
      'aria-label="' + escapeXml(label) + '"><title>' + escapeXml(label) + '</title>' +
      '<g fill="' + fill + '" stroke="' + INK + '" stroke-width="' + STROKE + '" stroke-linejoin="round" stroke-linecap="round">' +
      art + '</g>' +
      '<text x="0.5" y="0.55" font-size="0.3" text-anchor="middle" font-family="monospace" fill="' + INK + '" stroke="none">' +
      escapeXml(hintPhraseLabel(phrase)) + '</text></svg>';
  }

  function setHintError(message) {
    const box = $("hintError");
    const text = String(message || "").trim();
    box.textContent = text;
    box.hidden = text === "";
  }

  function hintOptionButton(phrase) {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "hint-option";
    button.dataset.phrase = phrase;
    button.setAttribute("aria-label", "Try " + phrase + " — one of four choices; the world decides what it does");
    const preview = document.createElement("span");
    preview.className = "hint-preview";
    preview.dataset.preview = "unknown";
    preview.innerHTML = hintPreviewSvg(phrase, "unknown");
    const label = document.createElement("span");
    label.className = "hint-phrase";
    label.textContent = phrase;
    button.appendChild(preview);
    button.appendChild(label);
    button.addEventListener("click", () => chooseHintOption(phrase));
    return button;
  }

  function renderHint(hint, meta) {
    if (!hint) state.lastHintChoice = "";
    const list = $("hintOptions");
    list.innerHTML = "";
    $("hintFeedback").textContent = "";
    $("hintFeedback").hidden = true;
    state.hint = null;

    if (!hint || !Array.isArray(hint.options) || !hint.options.length) {
      $("hintRationale").textContent = state.hintNotice
        ? "No four choices to show just now."
        : "No four choices yet — press the button when you want some.";
      $("hintPanel").dataset.source = "";
      $("hintPanel").dataset.options = "0";
      applyHintNotice();
      return;
    }

    const options = hint.options.slice(0, 4).map(String);
    state.hint = {
      placement: hint.placement && hint.placement.x != null && hint.placement.y != null
        ? { x: num(hint.placement.x, 0), y: num(hint.placement.y, 0) }
        : null,
      options: options,
      source: meta && meta.source ? String(meta.source) : "engine",
      model: meta && meta.model ? String(meta.model) : ""
    };
    $("hintRationale").textContent = String(hint.rationale || "One of these will do the job.") +
      (state.hint.model ? " (four choices from " + state.hint.model + ")" : "");
    $("hintPanel").dataset.source = state.hint.source;
    $("hintPanel").dataset.options = String(options.length);

    options.forEach((phrase) => list.appendChild(hintOptionButton(phrase)));
    // The previews are refined afterwards: a slow or missing lookup never delays a button.
    options.forEach((phrase) => refineHintPreview(phrase));

    // A note queued by a choice that could not be placed belongs under the fresh
    // four, not under the panel that was just cleared.
    applyHintNotice();
  }

  /* Shows a queued line under whichever set of choices is on screen (or under the
     empty panel), then forgets it. */
  function applyHintNotice() {
    if (!state.hintNotice) return;
    const box = $("hintFeedback");
    box.className = "hint-feedback is-try-again";
    box.hidden = false;
    box.textContent = state.hintNotice;
    state.hintNotice = "";
  }

  function findHintOption(phrase) {
    const buttons = $("hintOptions").querySelectorAll(".hint-option");
    for (let index = 0; index < buttons.length; index += 1) {
      if (buttons[index].dataset.phrase === phrase) return buttons[index];
    }
    return null;
  }

  function refineHintPreview(phrase) {
    enqueue(async () => {
      const result = await request("/api/words?q=" + encodeURIComponent(phrase) + "&limit=" + WORDS_LIMIT);
      const words = result.ok && result.data && Array.isArray(result.data.words) ? result.data.words : [];
      const wanted = String(phrase).toLowerCase();
      const known = words.some((word) => String(word).toLowerCase() === wanted);
      const match = result.ok && result.data && result.data.match ? result.data.match : null;
      const button = findHintOption(phrase);
      if (!button) return;
      const preview = button.querySelector(".hint-preview");
      if (!preview) return;
      preview.dataset.preview = known ? "known" : "improvised";
      if (match && match.shape) preview.dataset.shape = String(match.shape);
      preview.innerHTML = hintPreviewSvg(phrase, known ? "known" : "improvised",
        match ? match.shape : "", match ? match.color : "");
    }, null);
  }

  function chooseHintOption(phrase) {
    const hint = state.hint;
    if (!hint) return;
    const text = String(phrase || "").trim();
    if (!text) return;
    // The choice goes through the server so the engine can re-prove the spot for
    // the world as it stands now: the four choices were verified when they were
    // offered, and the player may have moved or drawn things since.
    enqueue(async () => {
      setSpawnNote("Trying “" + text + "”…", "quiet");
      const result = await request(
        ROUTES.worlds + "/" + encodeURIComponent(state.worldId) + "/hint/choose",
        { method: "POST", body: { phrase: text } }
      );

      if (!result.ok) {
        setSpawnNote("That did not work: " + result.error, "error");
        showError(result.error, () => chooseHintOption(text));
        // Four options that can no longer be placed would strand the player, so ask
        // for fresh ones for the world as it stands.
        state.hintNotice = "“" + text + "” could not be placed at the hint spot (" + result.error +
          "). Here are four fresh choices.";
        state.lastHintChoice = "";
        askForHint();
        return;
      }
      hideError();

      const data = result.data || {};
      const spawned = data.spawned || null;
      const approximate = !!data.approximate || !!(spawned && spawned.approximate);
      if (data.hint || state.world) applyState(data, data.events);
      state.notebook = state.notebook.concat([{
        phrase: text,
        key: String((spawned && spawned.key) || text),
        approximate: approximate,
        x: hint.placement ? hint.placement.x : 0,
        y: hint.placement ? hint.placement.y : 0
      }]);
      renderHistory();
      const note = data.note || (spawned && spawned.note) || "";
      setSpawnNote((note ? note + " " : "") + "“" + text + "” is in the world" +
        (data.movedSpot ? " (the four-choice spot was full, so it landed above you)." : "."),
        approximate ? "approximate" : "exact");
      $("notebookInput").value = "";

      const world = state.world || {};
      const goal = world.goal || {};
      const solvedNow = world.solved === true || goal.met === true;
      const line = solvedNow
        ? "“" + text + "” worked and the goal is met — the goal banner above has the rest."
        : "“" + text + "” did not reach the goal — try another of the four.";
      if (data.hint && data.hint.options && data.hint.options.length) {
        // Fresh choices arrive with their own line, so queue it for them: a line
        // written now would be wiped by the panel that renders them.
        state.hintNotice = line;
        state.lastHintChoice = solvedNow ? "" : text;
        renderHint(data.hint, { source: hint.source, model: hint.model });
      } else {
        hintChoiceFeedback(text);
      }
    }, () => chooseHintOption(text));
  }

  function hintChoiceFeedback(phrase) {
    const world = state.world || {};
    const goal = world.goal || {};
    const solved = world.solved === true || goal.met === true;
    const box = $("hintFeedback");
    box.className = "hint-feedback " + (solved ? "is-worked" : "is-try-again");
    box.hidden = false;
    box.textContent = solved
      ? "“" + phrase + "” worked and the goal is met — the goal banner above has the rest."
      : "“" + phrase + "” did not reach the goal — try another of the four.";
    // A choice often wins a tick or two later (the player climbs, the fire spreads),
    // so remember it and let refreshHintFeedback amend the line when it does.
    state.lastHintChoice = solved ? "" : phrase;
  }

  /* The judgement on a chosen option is only final once the world has settled: a
     choice that looked wrong as it landed can be the winning one two ticks later.
     This keeps the line honest instead of leaving "try another" under a solved goal. */
  function refreshHintFeedback() {
    if (!state.lastHintChoice) return;
    const world = state.world || {};
    const goal = world.goal || {};
    if (world.solved !== true && goal.met !== true) return;
    const phrase = state.lastHintChoice;
    state.lastHintChoice = "";
    state.hintNotice = "";
    const box = $("hintFeedback");
    if (box.hidden) return;
    box.className = "hint-feedback is-worked";
    box.textContent = "“" + phrase + "” worked and the goal is met — the goal banner above has the rest.";
  }

  function askForHint() {
    if (!state.world) {
      setHintError("There is no world open yet — pick a puzzle first.");
      return;
    }
    enqueue(async () => {
      setHintError("");
      $("hintButton").disabled = true;
      $("hintFeedback").hidden = true;
      $("hintRationale").textContent = "Working out four choices…";
      const result = await request(ROUTES.worlds + "/" + encodeURIComponent(state.worldId) + "/hint", { method: "POST", body: {} });
      $("hintButton").disabled = false;
      if (!result.ok) {
        $("hintRationale").textContent = "No four choices came back.";
        setHintError(result.error);
        // Four choices that the engine can no longer stand behind must not stay on
        // screen as clickable buttons: clear them and keep the reason visible.
        state.hintNotice = "No four choices right now: " + result.error;
        renderHint(null, null);
        return;
      }
      if (result.data && result.data.state) applyState(result.data.state, result.data.state.events);
      renderHint(result.data ? result.data.hint : null, { source: "engine" });
    }, askForHint);
  }

  function askAiHint() {
    if (!ai.enabled || !aiConfigured()) {
      setHintError(ai.enabled ? AI_NOT_CONFIGURED : AI_OFF_STATUS);
      return;
    }
    if (!state.world) {
      setHintError("There is no world open yet — pick a puzzle first.");
      return;
    }
    enqueue(async () => {
      setHintError("");
      $("aiHintButton").disabled = true;
      $("hintFeedback").hidden = true;
      $("hintRationale").textContent = "Asking " + ai.model + " for four choices…";
      const result = await request(AI_ROUTES.hint, {
        method: "POST",
        body: { baseUrl: ai.baseUrl, apiKey: ai.apiKey, model: ai.model, worldId: state.worldId }
      });
      $("aiHintButton").disabled = false;
      if (!result.ok) {
        $("hintRationale").textContent = "No four choices came back.";
        setHintError(result.error);
        renderHintAi();
        return;
      }
      if (result.data && result.data.state) applyState(result.data.state, result.data.state.events);
      renderHint(result.data ? result.data.hint : null, {
        source: result.data && result.data.usedAi === false ? "engine" : "ai",
        model: String((result.data && result.data.model) || ai.model)
      });
      renderHintAi();
    }, askAiHint);
  }

  function renderHintAi() {
    const usable = ai.enabled && aiConfigured() && !!state.world;
    $("aiHintButton").disabled = !usable;
    if (!ai.enabled) {
      $("aiHintNote").textContent = "AI is off, so this stays disabled — the engine's own four choices still work.";
      return;
    }
    if (!aiConfigured()) {
      $("aiHintNote").textContent = AI_NOT_CONFIGURED;
      return;
    }
    if (!state.world) {
      $("aiHintNote").textContent = "Open a puzzle first, then ask for four clever choices.";
      return;
    }
    $("aiHintNote").textContent = "Four choices would come from " + ai.model + ", and the world still decides what works.";
  }

  /* ------------------------------------------------------------------ SVG: the shape map */

  /* Clean, flat vector icons: one consistent outline, solid fills from the object's own colour,
   * an even margin inside every entity box, and the same visual weight for all twenty shapes so
   * the world reads as a single icon set. No gradients, no shadows, no filters, no loaded fonts.
   *
   * The block from INK down to the end of SHAPE_ART is deliberately self-contained: the Node
   * checks slice exactly this text out of app.js and evaluate it, so every shape can be proved to
   * return real SVG. Nothing in here may touch the DOM, the clock or a random number — the same
   * state must always draw exactly the same picture. */

  const INK = "#26323d";
  const STROKE = 0.05;       // one outline width for everything, in viewBox units

  function round2(value) {
    return Math.round(value * 1000) / 1000;
  }

  function shade(hex, amount) {
    const match = /^#?([0-9a-f]{6})$/i.exec(String(hex || "").trim());
    if (!match) return amount < 0 ? "#9a6b3a" : "#f0d9a8";
    const value = parseInt(match[1], 16);
    const parts = [(value >> 16) & 255, (value >> 8) & 255, value & 255].map((channel) => {
      const next = amount < 0 ? channel * (1 + amount) : channel + (255 - channel) * amount;
      return Math.round(Math.max(0, Math.min(255, next)));
    });
    return "#" + parts.map((channel) => channel.toString(16).padStart(2, "0")).join("");
  }

  function starPoints(cx, cy, outer, inner, points) {
    const total = Math.max(3, points) * 2;
    const list = [];
    for (let index = 0; index < total; index += 1) {
      const radius = index % 2 === 0 ? outer : inner;
      const angle = (Math.PI / Math.max(3, points)) * index - Math.PI / 2;
      list.push(round2(cx + Math.cos(angle) * radius) + "," + round2(cy + Math.sin(angle) * radius));
    }
    return list.join(" ");
  }

  // Two tidy, symmetric dots and a small smile.
  function face(cx, cy, size) {
    const radius = round2(0.032 * size);
    const eye = round2(0.075 * size);
    return '<circle cx="' + round2(cx - eye) + '" cy="' + round2(cy) + '" r="' + radius + '" fill="' + INK + '" stroke="none"/>' +
      '<circle cx="' + round2(cx + eye) + '" cy="' + round2(cy) + '" r="' + radius + '" fill="' + INK + '" stroke="none"/>' +
      '<path d="M' + round2(cx - 0.05 * size) + ' ' + round2(cy + 0.07 * size) + ' q ' + round2(0.05 * size) + ' ' + round2(0.06 * size) +
      ' ' + round2(0.1 * size) + ' 0" fill="none"/>';
  }

  /* One entry per pinned shape. Each returns a non-empty SVG fragment in local cell units, where
   * (0,0) is the entity's own top-left corner and (w,h) its size. An unknown shape falls back to
   * blob. Every shape keeps roughly a 0.06-cell margin and is centred in its box. */
  const SHAPE_ART = {
    box: (w, h, color) =>
      '<rect x="0.07" y="0.07" width="' + round2(w - 0.14) + '" height="' + round2(h - 0.14) + '" rx="0.09" fill="' + color + '"/>' +
      '<path d="M0.07 ' + round2(h * 0.32) + ' H' + round2(w - 0.07) + '" fill="none"/>' +
      '<path d="M' + round2(w * 0.5) + ' 0.07 V' + round2(h * 0.32) + '" fill="none"/>',

    ladder: (w, h, color) => {
      const rail = 0.12;
      const left = 0.12;
      const right = round2(w - 0.12 - rail);
      let out = '<rect x="' + left + '" y="0.09" width="' + rail + '" height="' + round2(h - 0.18) + '" rx="0.04" fill="' + color + '"/>' +
        '<rect x="' + right + '" y="0.09" width="' + rail + '" height="' + round2(h - 0.18) + '" rx="0.04" fill="' + color + '"/>';
      const rungs = h > 1.4 ? 4 : 3;
      for (let index = 1; index <= rungs; index += 1) {
        const y = round2(0.09 + (h - 0.18) * (index / (rungs + 1)) - 0.035);
        out += '<rect x="' + round2(left + rail - 0.02) + '" y="' + y + '" width="' + round2(right - left - rail + 0.04) +
          '" height="0.07" rx="0.03" fill="' + shade(color, 0.22) + '"/>';
      }
      return out;
    },

    rope: (w, h, color) => {
      const top = round2(h * 0.42);
      const bottom = round2(h * 0.58);
      return '<path fill="' + color + '" d="M0.1 ' + top + ' q0.2 -0.16 0.4 0 q0.2 0.16 0.4 0 L0.9 ' + bottom +
        ' q-0.2 0.16 -0.4 0 q-0.2 -0.16 -0.4 0 Z"/>' +
        '<circle cx="0.14" cy="' + round2(h * 0.38) + '" r="0.07" fill="' + shade(color, -0.2) + '"/>' +
        '<circle cx="' + round2(w - 0.14) + '" cy="' + round2(h * 0.62) + '" r="0.07" fill="' + shade(color, -0.2) + '"/>';
    },

    plank: (w, h, color) => {
      const top = round2(h * 0.3);
      const height = round2(h * 0.4);
      return '<rect x="0.06" y="' + top + '" width="' + round2(w - 0.12) + '" height="' + height + '" rx="0.05" fill="' + color + '"/>' +
        '<path d="M0.16 ' + round2(h * 0.42) + ' H' + round2(w - 0.16) + '" fill="none" opacity="0.55"/>' +
        '<path d="M0.16 ' + round2(h * 0.58) + ' H' + round2(w - 0.16) + '" fill="none" opacity="0.55"/>' +
        '<circle cx="0.18" cy="' + round2(h * 0.5) + '" r="0.035" fill="' + INK + '" stroke="none"/>' +
        '<circle cx="' + round2(w - 0.18) + '" cy="' + round2(h * 0.5) + '" r="0.035" fill="' + INK + '" stroke="none"/>';
    },

    blob: (w, h, color) => {
      const cx = round2(w / 2);
      const cy = round2(h / 2);
      return '<path fill="' + color + '" d="M' + cx + ' 0.08' +
        ' C ' + round2(w * 0.82) + ' 0.08 ' + round2(w * 0.94) + ' ' + round2(h * 0.28) + ' ' + round2(w * 0.92) + ' ' + cy +
        ' C ' + round2(w * 0.9) + ' ' + round2(h * 0.86) + ' ' + round2(w * 0.72) + ' ' + round2(h * 0.94) + ' ' + cx + ' ' + round2(h * 0.94) +
        ' C ' + round2(w * 0.28) + ' ' + round2(h * 0.94) + ' ' + round2(w * 0.1) + ' ' + round2(h * 0.86) + ' ' + round2(w * 0.08) + ' ' + cy +
        ' C ' + round2(w * 0.06) + ' ' + round2(h * 0.28) + ' ' + round2(w * 0.18) + ' 0.08 ' + cx + ' 0.08 Z"/>' +
        '<circle cx="' + round2(w * 0.36) + '" cy="' + round2(h * 0.34) + '" r="0.05" fill="#ffffff" stroke="none" opacity="0.75"/>';
    },

    circle: (w, h, color) => {
      const radius = round2(Math.min(w, h) / 2 - 0.09);
      return '<circle cx="' + round2(w / 2) + '" cy="' + round2(h / 2) + '" r="' + radius + '" fill="' + color + '"/>' +
        '<circle cx="' + round2(w * 0.36) + '" cy="' + round2(h * 0.34) + '" r="0.05" fill="#ffffff" stroke="none" opacity="0.75"/>';
    },

    star: (w, h, color) => {
      const outer = round2(Math.min(w, h) / 2 - 0.08);
      return '<polygon points="' + starPoints(w / 2, h / 2, outer, round2(Math.min(w, h) * 0.21), 5) + '" fill="' + color + '"/>';
    },

    tree: (w, h, color) => {
      const trunkW = round2(w * 0.16);
      const canopy = round2(Math.min(w, h) * 0.3);
      return '<rect x="' + round2(w / 2 - trunkW / 2) + '" y="' + round2(h * 0.52) + '" width="' + trunkW + '" height="' + round2(h * 0.42) +
        '" rx="0.03" fill="#9a6b3a"/>' +
        '<circle cx="' + round2(w * 0.5) + '" cy="' + round2(h * 0.34) + '" r="' + canopy + '" fill="' + color + '"/>' +
        '<circle cx="' + round2(w * 0.26) + '" cy="' + round2(h * 0.48) + '" r="' + round2(canopy * 0.68) + '" fill="' + color + '"/>' +
        '<circle cx="' + round2(w * 0.74) + '" cy="' + round2(h * 0.48) + '" r="' + round2(canopy * 0.68) + '" fill="' + color + '"/>';
    },

    flame: (w, h, color) =>
      '<path class="flame-lick" fill="' + color + '" d="M' + round2(w * 0.5) + ' 0.08' +
      ' C ' + round2(w * 0.74) + ' ' + round2(h * 0.34) + ' ' + round2(w * 0.9) + ' ' + round2(h * 0.54) + ' ' + round2(w * 0.82) + ' ' + round2(h * 0.76) +
      ' C ' + round2(w * 0.74) + ' ' + round2(h * 0.92) + ' ' + round2(w * 0.26) + ' ' + round2(h * 0.92) + ' ' + round2(w * 0.18) + ' ' + round2(h * 0.76) +
      ' C ' + round2(w * 0.1) + ' ' + round2(h * 0.54) + ' ' + round2(w * 0.26) + ' ' + round2(h * 0.34) + ' ' + round2(w * 0.5) + ' 0.08 Z"/>' +
      '<path fill="#ffd24d" stroke="none" d="M' + round2(w * 0.5) + ' ' + round2(h * 0.36) +
      ' C ' + round2(w * 0.62) + ' ' + round2(h * 0.52) + ' ' + round2(w * 0.64) + ' ' + round2(h * 0.72) + ' ' + round2(w * 0.5) + ' ' + round2(h * 0.84) +
      ' C ' + round2(w * 0.36) + ' ' + round2(h * 0.72) + ' ' + round2(w * 0.38) + ' ' + round2(h * 0.52) + ' ' + round2(w * 0.5) + ' ' + round2(h * 0.36) + ' Z"/>',

    key: (w, h, color) => {
      const bow = round2(Math.min(w, h) * 0.2);
      const cy = round2(h * 0.5);
      return '<circle cx="' + round2(w * 0.25) + '" cy="' + cy + '" r="' + bow + '" fill="' + color + '"/>' +
        '<circle cx="' + round2(w * 0.25) + '" cy="' + cy + '" r="' + round2(bow * 0.42) + '" fill="#ffffff" stroke="none" opacity="0.85"/>' +
        '<rect x="' + round2(w * 0.41) + '" y="' + round2(cy - 0.045) + '" width="' + round2(w * 0.51) + '" height="0.09" rx="0.02" fill="' + color + '"/>' +
        '<rect x="' + round2(w * 0.78) + '" y="' + cy + '" width="0.07" height="0.17" rx="0.02" fill="' + color + '"/>' +
        '<rect x="' + round2(w * 0.66) + '" y="' + cy + '" width="0.07" height="0.15" rx="0.02" fill="' + color + '"/>' +
        '<rect x="' + round2(w * 0.9) + '" y="' + cy + '" width="0.06" height="0.12" rx="0.02" fill="' + color + '"/>';
    },

    tool: (w, h, color) =>
      '<rect x="' + round2(w * 0.46) + '" y="' + round2(h * 0.3) + '" width="0.1" height="' + round2(h * 0.64) + '" rx="0.03" fill="#c98a4b"/>' +
      '<rect x="' + round2(w * 0.16) + '" y="' + round2(h * 0.12) + '" width="' + round2(w * 0.68) + '" height="' + round2(h * 0.2) +
      '" rx="0.05" fill="' + color + '"/>' +
      '<path d="M' + round2(w * 0.16) + ' ' + round2(h * 0.16) + ' q-0.09 0.05 0 0.1" fill="none"/>' +
      '<path d="M' + round2(w * 0.51) + ' ' + round2(h * 0.32) + ' V' + round2(h * 0.86) + '" fill="none" opacity="0.5"/>',

    animal: (w, h, color) => {
      const bodyCY = round2(h * 0.62);
      const bodyRX = round2(w * 0.33);
      const bodyRY = round2(h * 0.21);
      return '<path d="M' + round2(w * 0.16) + ' ' + round2(bodyCY - bodyRY * 0.5) + ' q-0.12 -0.14 -0.02 -0.22" fill="none"/>' +
        '<ellipse cx="' + round2(w * 0.44) + '" cy="' + bodyCY + '" rx="' + bodyRX + '" ry="' + bodyRY + '" fill="' + color + '"/>' +
        '<path d="M' + round2(w * 0.3) + ' ' + round2(bodyCY + bodyRY - 0.02) + ' V' + round2(h * 0.94) + '" fill="none"/>' +
        '<path d="M' + round2(w * 0.44) + ' ' + round2(bodyCY + bodyRY - 0.02) + ' V' + round2(h * 0.94) + '" fill="none"/>' +
        '<path d="M' + round2(w * 0.58) + ' ' + round2(bodyCY + bodyRY - 0.02) + ' V' + round2(h * 0.94) + '" fill="none"/>' +
        '<circle cx="' + round2(w * 0.74) + '" cy="' + round2(h * 0.38) + '" r="' + round2(Math.min(w, h) * 0.2) + '" fill="' + color + '"/>' +
        '<path d="M' + round2(w * 0.65) + ' ' + round2(h * 0.24) + ' l-0.03 -0.16 l0.14 0.07 Z" fill="' + color + '"/>' +
        '<path d="M' + round2(w * 0.83) + ' ' + round2(h * 0.24) + ' l0.03 -0.16 l-0.14 0.07 Z" fill="' + color + '"/>' +
        face(w * 0.74, h * 0.38, Math.min(w, h) * 0.8);
    },

    person: (w, h, color) => {
      const headR = round2(Math.min(w, h) * 0.19);
      const headCY = round2(h * 0.23);
      return '<circle cx="' + round2(w / 2) + '" cy="' + headCY + '" r="' + headR + '" fill="#f4d3ac"/>' +
        face(w / 2, headCY, Math.min(w, h) * 0.85) +
        '<rect x="' + round2(w * 0.33) + '" y="' + round2(h * 0.46) + '" width="' + round2(w * 0.34) + '" height="' + round2(h * 0.3) +
        '" rx="0.06" fill="' + color + '"/>' +
        '<path d="M' + round2(w * 0.33) + ' ' + round2(h * 0.54) + ' L' + round2(w * 0.13) + ' ' + round2(h * 0.68) + '" fill="none"/>' +
        '<path d="M' + round2(w * 0.67) + ' ' + round2(h * 0.54) + ' L' + round2(w * 0.87) + ' ' + round2(h * 0.68) + '" fill="none"/>' +
        '<path d="M' + round2(w * 0.43) + ' ' + round2(h * 0.76) + ' L' + round2(w * 0.35) + ' ' + round2(h * 0.95) + '" fill="none"/>' +
        '<path d="M' + round2(w * 0.57) + ' ' + round2(h * 0.76) + ' L' + round2(w * 0.65) + ' ' + round2(h * 0.95) + '" fill="none"/>';
    },

    bottle: (w, h, color) =>
      '<rect x="' + round2(w * 0.43) + '" y="0.08" width="' + round2(w * 0.14) + '" height="' + round2(h * 0.18) + '" rx="0.03" fill="' +
      shade(color, -0.12) + '"/>' +
      '<rect x="' + round2(w * 0.28) + '" y="' + round2(h * 0.24) + '" width="' + round2(w * 0.44) + '" height="' + round2(h * 0.68) +
      '" rx="0.08" fill="' + color + '"/>' +
      '<path d="M' + round2(w * 0.3) + ' ' + round2(h * 0.58) + ' H' + round2(w * 0.7) + ' V' + round2(h * 0.88) +
      '" fill="#8ed0ff" stroke="none" opacity="0.75"/>',

    book: (w, h, color) =>
      '<rect x="0.07" y="' + round2(h * 0.15) + '" width="' + round2(w - 0.14) + '" height="' + round2(h * 0.7) + '" rx="0.05" fill="' + color + '"/>' +
      '<rect x="' + round2(w * 0.19) + '" y="' + round2(h * 0.23) + '" width="' + round2(w * 0.68) + '" height="' + round2(h * 0.54) +
      '" rx="0.03" fill="#fdf6e3"/>' +
      '<path d="M' + round2(w * 0.27) + ' ' + round2(h * 0.39) + ' H' + round2(w * 0.79) + '" fill="none" opacity="0.5"/>' +
      '<path d="M' + round2(w * 0.27) + ' ' + round2(h * 0.53) + ' H' + round2(w * 0.73) + '" fill="none" opacity="0.5"/>' +
      '<path d="M' + round2(w * 0.19) + ' ' + round2(h * 0.23) + ' V' + round2(h * 0.77) + '" fill="none"/>',

    vehicle: (w, h, color) => {
      const wheelY = round2(h * 0.8);
      const wheelR = round2(Math.min(w, h) * 0.12);
      return '<rect x="0.08" y="' + round2(h * 0.42) + '" width="' + round2(w - 0.16) + '" height="' + round2(h * 0.32) + '" rx="0.07" fill="' +
        color + '"/>' +
        '<path fill="' + shade(color, 0.22) + '" d="M' + round2(w * 0.26) + ' ' + round2(h * 0.42) + ' L' + round2(w * 0.34) + ' ' + round2(h * 0.22) +
        ' H' + round2(w * 0.62) + ' L' + round2(w * 0.7) + ' ' + round2(h * 0.42) + ' Z"/>' +
        '<circle cx="' + round2(w * 0.32) + '" cy="' + wheelY + '" r="' + wheelR + '" fill="#3a4450"/>' +
        '<circle cx="' + round2(w * 0.7) + '" cy="' + wheelY + '" r="' + wheelR + '" fill="#3a4450"/>' +
        '<circle cx="' + round2(w * 0.87) + '" cy="' + round2(h * 0.54) + '" r="0.04" fill="#ffe066" stroke="none"/>';
    },

    chest: (w, h, color, entity) => {
      const open = !!(entity && entity.opened);
      const body = '<rect x="0.08" y="' + round2(h * 0.44) + '" width="' + round2(w - 0.16) + '" height="' + round2(h * 0.48) +
        '" rx="0.05" fill="' + color + '"/>' +
        '<rect x="' + round2(w * 0.43) + '" y="' + round2(h * 0.58) + '" width="' + round2(w * 0.14) + '" height="' + round2(h * 0.18) +
        '" rx="0.02" fill="#ffd24d"/>' +
        '<path d="M' + round2(w * 0.3) + ' ' + round2(h * 0.44) + ' V' + round2(h * 0.92) + '" fill="none" opacity="0.45"/>' +
        '<path d="M' + round2(w * 0.7) + ' ' + round2(h * 0.44) + ' V' + round2(h * 0.92) + '" fill="none" opacity="0.45"/>';
      const lid = open
        ? '<path fill="' + shade(color, 0.22) + '" d="M0.08 ' + round2(h * 0.44) + ' L0.14 ' + round2(h * 0.1) + ' H' + round2(w * 0.86) +
          ' L' + round2(w - 0.08) + ' ' + round2(h * 0.44) + ' Z"/>'
        : '<rect x="0.08" y="' + round2(h * 0.26) + '" width="' + round2(w - 0.16) + '" height="' + round2(h * 0.2) +
          '" rx="0.05" fill="' + shade(color, 0.22) + '"/>';
      return lid + body;
    },

    candle: (w, h, color, entity) => {
      const lit = !!(entity && entity.burning);
      return '<rect x="' + round2(w * 0.34) + '" y="' + round2(h * 0.36) + '" width="' + round2(w * 0.32) + '" height="' + round2(h * 0.6) +
        '" rx="0.04" fill="' + color + '"/>' +
        '<path d="M' + round2(w * 0.42) + ' ' + round2(h * 0.36) + ' V' + round2(h * 0.96) + '" fill="none" opacity="0.45"/>' +
        '<path d="M' + round2(w * 0.5) + ' ' + round2(h * 0.36) + ' V' + round2(h * 0.28) + '" fill="none"/>' +
        (lit
          ? '<path class="flame-lick" fill="#ffb347" d="M' + round2(w * 0.5) + ' ' + round2(h * 0.06) +
            ' C ' + round2(w * 0.64) + ' ' + round2(h * 0.16) + ' ' + round2(w * 0.62) + ' ' + round2(h * 0.26) + ' ' + round2(w * 0.5) + ' ' + round2(h * 0.28) +
            ' C ' + round2(w * 0.38) + ' ' + round2(h * 0.26) + ' ' + round2(w * 0.36) + ' ' + round2(h * 0.16) + ' ' + round2(w * 0.5) + ' ' + round2(h * 0.06) + ' Z"/>' +
            '<path class="flame-lick" fill="#ffd24d" stroke="none" d="M' + round2(w * 0.5) + ' ' + round2(h * 0.12) +
            ' C ' + round2(w * 0.57) + ' ' + round2(h * 0.19) + ' ' + round2(w * 0.56) + ' ' + round2(h * 0.26) + ' ' + round2(w * 0.5) + ' ' + round2(h * 0.27) +
            ' C ' + round2(w * 0.44) + ' ' + round2(h * 0.26) + ' ' + round2(w * 0.43) + ' ' + round2(h * 0.19) + ' ' + round2(w * 0.5) + ' ' + round2(h * 0.12) + ' Z"/>'
          : '');
    },

    flag: (w, h, color) =>
      '<path d="M' + round2(w * 0.2) + ' 0.07 V' + round2(h * 0.94) + '" fill="none"/>' +
      '<path fill="' + color + '" d="M' + round2(w * 0.23) + ' 0.1 H' + round2(w * 0.9) + ' L' + round2(w * 0.74) + ' ' + round2(h * 0.28) +
      ' L' + round2(w * 0.9) + ' ' + round2(h * 0.46) + ' H' + round2(w * 0.23) + ' Z"/>' +
      '<circle cx="' + round2(w * 0.2) + '" cy="0.07" r="0.05" fill="' + color + '"/>',

    machine: (w, h, color) => {
      const dialCY = round2(h * 0.68);
      const dialR = round2(Math.min(w, h) * 0.11);
      return '<rect x="0.07" y="0.07" width="' + round2(w - 0.14) + '" height="' + round2(h - 0.14) + '" rx="0.07" fill="' + color + '"/>' +
        '<rect x="' + round2(w * 0.18) + '" y="' + round2(h * 0.18) + '" width="' + round2(w * 0.64) + '" height="' + round2(h * 0.22) +
        '" rx="0.03" fill="#cfe9ff"/>' +
        '<path d="M' + round2(w * 0.24) + ' ' + round2(h * 0.29) + ' H' + round2(w * 0.7) + '" fill="none" opacity="0.5"/>' +
        '<circle cx="' + round2(w * 0.32) + '" cy="' + dialCY + '" r="' + dialR + '" fill="#ffd24d"/>' +
        '<path d="M' + round2(w * 0.32) + ' ' + dialCY + ' L' + round2(w * 0.37) + ' ' + round2(dialCY - dialR * 0.6) + '" fill="none" stroke-width="0.04"/>' +
        '<circle cx="' + round2(w * 0.62) + '" cy="' + round2(h * 0.62) + '" r="0.05" fill="#7fd6a6"/>' +
        '<circle cx="' + round2(w * 0.62) + '" cy="' + round2(h * 0.78) + '" r="0.05" fill="#e8443a"/>' +
        '<path d="M' + round2(w * 0.68) + ' ' + round2(h * 0.78) + ' H' + round2(w * 0.84) + '" fill="none" opacity="0.5"/>';
    }
  };

  /* ------------------------------------------------------------------ SVG: terrain */

  function grassTicks(x, y) {
    let out = "";
    [0.22, 0.5, 0.78].forEach((offset) => {
      out += '<path d="M' + round2(x + offset) + ' ' + round2(y + 0.2) + ' V' + round2(y + 0.06) + '" fill="none" stroke-width="0.04"/>';
    });
    return out;
  }

  // One cell of scenery, from the terrain id alone. Every value is a constant, so the same world
  // always draws the same picture.
  function terrainCell(world, x, y) {
    const width = Math.max(1, num(world.width, 1));
    const value = num(world.terrain[y * width + x], 0);
    if (!value) return "";
    const above = y === 0 ? 0 : num(world.terrain[(y - 1) * width + x], 0);
    const same = above === value;
    const surface = same ? "" :
      '<path d="M' + x + ' ' + round2(y + 0.025) + ' H' + (x + 1) + '" fill="none" stroke="' + INK + '" stroke-width="' + STROKE + '"/>';

    if (value === 1) {
      return '<rect x="' + x + '" y="' + y + '" width="1" height="1" fill="#c98a4b"/>' +
        (same ? "" : '<rect x="' + x + '" y="' + y + '" width="1" height="0.22" fill="#6cbf5a"/>' + grassTicks(x, y)) +
        '<circle cx="' + round2(x + 0.3) + '" cy="' + round2(y + 0.62) + '" r="0.04" fill="#b3743a" stroke="none"/>' +
        '<circle cx="' + round2(x + 0.68) + '" cy="' + round2(y + 0.78) + '" r="0.035" fill="#b3743a" stroke="none"/>' +
        surface;
    }

    if (value === 2) {
      return '<rect x="' + x + '" y="' + y + '" width="1" height="1" fill="#7cc4f2"/>' +
        (same ? "" : '<rect x="' + x + '" y="' + y + '" width="1" height="0.18" fill="#a8dcff"/>') +
        '<path d="M' + round2(x + 0.12) + ' ' + round2(y + 0.42) + ' q0.19 -0.08 0.38 0 q0.19 0.08 0.38 0" fill="none" stroke="#4a9ad4" stroke-width="0.045"/>' +
        '<path d="M' + round2(x + 0.12) + ' ' + round2(y + 0.68) + ' q0.19 -0.08 0.38 0 q0.19 0.08 0.38 0" fill="none" stroke="#4a9ad4" stroke-width="0.045"/>' +
        surface;
    }

    if (value === 3) {
      let out = '<rect x="' + x + '" y="' + y + '" width="1" height="1" fill="#9a6b3a"/>' +
        '<path d="M' + round2(x + 0.36) + ' ' + y + ' V' + round2(y + 1) + '" fill="none" stroke="#7d5430" stroke-width="0.04"/>' +
        '<path d="M' + round2(x + 0.64) + ' ' + y + ' V' + round2(y + 1) + '" fill="none" stroke="#7d5430" stroke-width="0.04"/>';
      if (!same) {
        out += '<circle cx="' + round2(x + 0.5) + '" cy="' + round2(y - 0.04) + '" r="0.4" fill="#55b04a"/>' +
          '<circle cx="' + round2(x + 0.22) + '" cy="' + round2(y + 0.08) + '" r="0.24" fill="#55b04a"/>' +
          '<circle cx="' + round2(x + 0.78) + '" cy="' + round2(y + 0.08) + '" r="0.24" fill="#55b04a"/>';
      }
      return out + surface;
    }

    if (value === 4) {
      const row = y % 2;
      return '<rect x="' + x + '" y="' + y + '" width="1" height="1" fill="#dfe4ea"/>' +
        '<path d="M' + x + ' ' + round2(y + 0.5) + ' H' + (x + 1) + '" fill="none" stroke="#b9c0ca" stroke-width="0.045"/>' +
        '<path d="M' + round2(x + (row ? 0.33 : 0.67)) + ' ' + y + ' V' + round2(y + 0.5) + '" fill="none" stroke="#b9c0ca" stroke-width="0.045"/>' +
        '<path d="M' + round2(x + (row ? 0.67 : 0.33)) + ' ' + round2(y + 0.5) + ' V' + round2(y + 1) + '" fill="none" stroke="#b9c0ca" stroke-width="0.045"/>' +
        '<rect x="' + x + '" y="' + y + '" width="1" height="1" fill="none" stroke="' + INK + '" stroke-width="0.03"/>' +
        surface;
    }

    if (value === 5) {
      return '<rect x="' + x + '" y="' + y + '" width="1" height="1" fill="#c9c6c1"/>' +
        [[0.3, 0.3], [0.7, 0.3], [0.3, 0.7], [0.7, 0.7]].map((dot) =>
          '<circle cx="' + round2(x + dot[0]) + '" cy="' + round2(y + dot[1]) + '" r="0.045" fill="#a5a29d" stroke="none"/>'
        ).join("") +
        surface;
    }

    return '<rect x="' + x + '" y="' + y + '" width="1" height="1" fill="#dfe4ea"/>' + surface;
  }

  /* ------------------------------------------------------------------ SVG: entities and the player */

  function escapeXml(value) {
    return String(value == null ? "" : value)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");
  }

  function cleanColor(value) {
    return typeof value === "string" && /^#[0-9a-f]{3,8}$/i.test(value.trim()) ? value.trim() : "";
  }

  function entityArt(entity) {
    const object = entity && entity.object ? entity.object : {};
    const x = num(entity && entity.x, 0);
    const y = num(entity && entity.y, 0);
    const w = Math.max(1, num(entity && entity.w, 1));
    const h = Math.max(1, num(entity && entity.h, 1));
    const color = cleanColor(object.color) || "#c98a3c";
    const shape = Object.prototype.hasOwnProperty.call(SHAPE_ART, object.shape) ? String(object.shape) : "blob";
    const art = SHAPE_ART[shape] || SHAPE_ART.blob;

    const tags = Array.isArray(object.tags) ? object.tags : [];
    const flags = [];
    if (entity.burning) flags.push("on fire");
    if (entity.opened) flags.push("open");
    if (entity.cut) flags.push("cut");
    if (entity.powered) flags.push("powered");
    if (entity.held) flags.push("in hand");
    const name = String(object.name || object.noun || object.key || "something");
    const title = name + " — " +
      (tags.length ? tags.join(", ") : "no particular tags") +
      (flags.length ? ", " + flags.join(", ") : "");

    const classes = ["entity"];
    if (entity.cut) classes.push("is-cut");
    if (entity.burning) classes.push("is-burning");
    if (entity.powered) classes.push("is-powered");

    let out = '<g class="' + classes.join(" ") + '"' +
      ' data-entity-id="' + escapeXml(num(entity.id, 0)) + '"' +
      ' data-key="' + escapeXml(object.key || "") + '"' +
      ' data-shape="' + escapeXml(shape) + '"' +
      ' data-tags="' + escapeXml(tags.join(",")) + '"' +
      ' data-burning="' + (entity.burning ? "true" : "false") + '"' +
      ' data-opened="' + (entity.opened ? "true" : "false") + '"' +
      ' data-cut="' + (entity.cut ? "true" : "false") + '"' +
      ' data-powered="' + (entity.powered ? "true" : "false") + '"' +
      ' data-held="' + (entity.held ? "true" : "false") + '"' +
      (entity.cut ? ' opacity="0.5"' : "") +
      ' transform="translate(' + round2(x) + ' ' + round2(y) + ')">' +
      '<title>' + escapeXml(title) + '</title>';

    if (entity.powered) {
      out += '<circle class="glow" cx="' + round2(w / 2) + '" cy="' + round2(h / 2) + '" r="' + round2(Math.min(w, h) / 2 - 0.02) +
        '" fill="none" stroke="#ffd24d" stroke-width="0.06"/>';
    }
    out += '<g class="art" fill="' + color + '" stroke="' + INK + '" stroke-width="' + STROKE +
      '" stroke-linejoin="round" stroke-linecap="round">' + art(w, h, color, entity) + '</g>';

    if (entity.burning) {
      out += '<g class="burning-art" fill="#ff8a3c" stroke="' + INK + '" stroke-width="' + STROKE + '" stroke-linejoin="round">' +
        '<path class="flame-lick" d="M' + round2(w * 0.3) + ' ' + round2(h * 0.24) +
        ' C ' + round2(w * 0.42) + ' ' + round2(h * 0.08) + ' ' + round2(w * 0.46) + ' ' + round2(h * 0.16) + ' ' + round2(w * 0.38) + ' ' + round2(h * 0.3) + ' Z"/>' +
        '<path class="flame-lick" fill="#ffd24d" d="M' + round2(w * 0.66) + ' ' + round2(h * 0.2) +
        ' C ' + round2(w * 0.78) + ' ' + round2(h * 0.04) + ' ' + round2(w * 0.82) + ' ' + round2(h * 0.12) + ' ' + round2(w * 0.74) + ' ' + round2(h * 0.26) + ' Z"/>' +
        '</g>';
    }
    if (entity.cut) {
      out += '<path class="cut-line" d="M0.06 ' + round2(h - 0.06) + ' L' + round2(w - 0.06) + ' 0.06" fill="none" stroke="#e8443a" ' +
        'stroke-width="0.05" stroke-dasharray="0.1 0.07"/>';
    }
    if (object.approximate) {
      out += '<rect class="improvised" x="0.04" y="0.04" width="' + round2(w - 0.08) + '" height="' + round2(h - 0.08) +
        '" rx="0.06" fill="none" stroke="#f0a500" stroke-width="0.045" stroke-dasharray="0.1 0.08"/>';
    }

    const glyph = String(object.glyph || "").trim();
    if (glyph) {
      out += '<text class="glyph" x="' + round2(w / 2) + '" y="' + round2(h / 2) + '" font-size="0.45" text-anchor="middle" ' +
        'dominant-baseline="middle" font-family="monospace" fill="' + INK + '" stroke="none">' + escapeXml(glyph) + '</text>';
    }
    return out + '</g>';
  }

  // A friendly little character: round head, dot eyes, a smile, a body and a notebook.
  // Flipped with scale(-1 1) when they walk left.
  function playerArt(player, entities) {
    const x = num(player.x, 0);
    const y = num(player.y, 0);
    const left = state.facing === "left";
    const holdingId = num(player.holding, 0);
    const held = holdingId === 0 ? null : entities.filter((entity) => num(entity.id, -1) === holdingId)[0];

    let art = '<circle cx="0.5" cy="0.24" r="0.19" fill="#f4d3ac"/>' +
      '<circle cx="0.43" cy="0.22" r="0.028" fill="' + INK + '" stroke="none"/>' +
      '<circle cx="0.57" cy="0.22" r="0.028" fill="' + INK + '" stroke="none"/>' +
      '<path d="M0.44 0.3 q0.06 0.06 0.12 0" fill="none" stroke="' + INK + '" stroke-width="0.035"/>' +
      '<rect x="0.38" y="0.44" width="0.24" height="0.32" rx="0.07" fill="#4f86d6"/>' +
      '<path d="M0.4 0.52 L0.28 0.66" fill="none" stroke="' + INK + '" stroke-width="0.045"/>' +
      '<path d="M0.6 0.52 L0.7 0.64" fill="none" stroke="' + INK + '" stroke-width="0.045"/>' +
      '<path d="M0.44 0.76 V0.94" fill="none" stroke="' + INK + '" stroke-width="0.05"/>' +
      '<path d="M0.56 0.76 V0.94" fill="none" stroke="' + INK + '" stroke-width="0.05"/>' +
      '<rect x="0.7" y="0.54" width="0.24" height="0.28" rx="0.03" fill="#fdf6e3"/>' +
      '<path d="M0.75 0.64 H0.89" fill="none" stroke="' + INK + '" stroke-width="0.025"/>' +
      '<path d="M0.75 0.73 H0.85" fill="none" stroke="' + INK + '" stroke-width="0.025"/>';

    if (held && held.object) {
      art += '<rect x="0.14" y="0.46" width="0.22" height="0.22" rx="0.04" fill="' + (cleanColor(held.object.color) || "#c98a3c") +
        '" stroke="' + INK + '" stroke-width="0.045"/>';
    }

    const flip = left ? ' scale(-1 1) translate(-1 0)' : "";
    return '<g id="player" data-x="' + round2(x) + '" data-y="' + round2(y) + '" data-facing="' + (left ? "left" : "right") + '"' +
      ' data-on-ground="' + (player.onGround === false ? "false" : "true") + '"' +
      ' transform="translate(' + round2(x) + ' ' + round2(y) + ')" stroke-linejoin="round" stroke-linecap="round">' +
      '<g transform="scale(0.94 0.94)' + flip + '">' + art + '</g></g>';
  }

  /* ------------------------------------------------------------------ SVG: the world layer */

  function worldLabel(world) {
    const puzzle = world.puzzle || state.puzzle || {};
    const count = Array.isArray(world.entities) ? world.entities.length : 0;
    const title = String(puzzle.title || (state.aiPuzzle && state.aiPuzzle.spec ? state.aiPuzzle.spec.title : "") || "The world");
    return title + ", " + num(world.width, 1) + " by " + num(world.height, 1) + " cells, " +
      count + (count === 1 ? " object" : " objects") + (world.solved === true ? ", solved" : "");
  }

  function renderWorldSvg(world) {
    const svg = $("worldSvg");
    const width = Math.max(1, num(world.width, 1));
    const height = Math.max(1, num(world.height, 1));
    const entities = Array.isArray(world.entities) ? world.entities : [];

    svg.setAttribute("viewBox", "0 0 " + width + " " + height);
    svg.setAttribute("preserveAspectRatio", "xMidYMid meet");
    svg.setAttribute("aria-label", worldLabel(world));
    svg.style.aspectRatio = width + " / " + height;

    let terrain = '<rect class="sky" x="0" y="0" width="' + width + '" height="' + height + '" fill="#dbeafe" stroke="none"/>';
    for (let y = 0; y < height; y += 1) {
      for (let x = 0; x < width; x += 1) terrain += terrainCell(world, x, y);
    }
    $("terrainLayer").innerHTML = terrain;
    $("entityLayer").innerHTML = entities.map(entityArt).join("");
    $("playerLayer").innerHTML = playerArt(world.player || { x: 0, y: 0, onGround: true }, entities);

    const marker = $("placementMarker");
    const target = placementCell(world);
    marker.setAttribute("x", target.x);
    marker.setAttribute("y", target.y);
    marker.setAttribute("width", 1);
    marker.setAttribute("height", 1);
    marker.setAttribute("data-source", target.source);
    marker.setAttribute("stroke", target.source === "chosen" ? "#e8443a" : "#1f7a3d");
    // An SVG element has no `hidden` property, so the attribute is the only honest switch.
    marker.removeAttribute("hidden");

    const player = world.player || {};
    svg.dataset.ticks = String(num(world.ticks, 0));
    svg.dataset.playerX = String(num(player.x, 0));
    svg.dataset.playerY = String(num(player.y, 0));
    svg.dataset.objects = String(entities.length);
    svg.dataset.solved = world.solved === true ? "true" : "false";
  }

  function cellFromPointer(event) {
    const svg = $("worldSvg");
    const world = state.world;
    if (!world) return null;
    const rect = svg.getBoundingClientRect();
    if (!rect.width || !rect.height) return null;
    const box = svg.viewBox.baseVal;
    const vw = box && box.width ? box.width : Math.max(1, num(world.width, 1));
    const vh = box && box.height ? box.height : Math.max(1, num(world.height, 1));
    // preserveAspectRatio="xMidYMid meet" letterboxes the drawing, so map through the content box.
    const scale = Math.min(rect.width / vw, rect.height / vh);
    const offsetX = (rect.width - vw * scale) / 2;
    const offsetY = (rect.height - vh * scale) / 2;
    const vx = (event.clientX - rect.left - offsetX) / scale;
    const vy = (event.clientY - rect.top - offsetY) / scale;
    return {
      x: clamp(Math.floor(vx), 0, Math.max(0, vw - 1)),
      y: clamp(Math.floor(vy), 0, Math.max(0, vh - 1))
    };
  }

  function onSvgClick(event) {
    const target = cellFromPointer(event);
    if (!target) return;
    state.placement = target;
    renderPlacement(state.world);
    renderWorldSvg(state.world);
    $("worldSvg").focus();
  }

  /* ------------------------------------------------------------------ wiring */

  function wire() {
    $("notebookForm").addEventListener("submit", onSubmitNotebook);
    $("notebookInput").addEventListener("input", onNotebookInput);
    $("notebookInput").addEventListener("blur", () => {
      window.setTimeout(() => {
        if (state.suggestions.length) {
          state.suggestions = [];
          state.suggestionIndex = -1;
          renderSuggestions();
        }
      }, 150);
    });

    $("btnLeft").addEventListener("click", () => doAction("left"));
    $("btnRight").addEventListener("click", () => doAction("right"));
    $("btnJump").addEventListener("click", () => doAction("jump"));
    $("btnTake").addEventListener("click", () => doAction("take"));
    $("btnDrop").addEventListener("click", () => doAction("drop"));
    $("btnStep").addEventListener("click", advanceTime);
    $("btnReset").addEventListener("click", resetPuzzle);
    $("btnDropFromSky").addEventListener("click", dropFromSky);
    $("newPuzzleButton").addEventListener("click", chooseAnotherPuzzle);
    $("puzzleHintButton").addEventListener("click", toggleHint);
    $("errorRetry").addEventListener("click", () => {
      const retry = state.retry || boot;
      hideError();
      retry();
    });

    // AI settings: one key, one store, and an explicit reveal for the password field.
    $("aiSettingsForm").addEventListener("submit", saveAiSettings);
    $("aiToggle").addEventListener("change", () => {
      ai.enabled = $("aiToggle").checked;
      persistAi();
      renderAi();
    });
    $("aiKeyReveal").addEventListener("click", toggleKeyReveal);
    $("aiForgetButton").addEventListener("click", forgetAi);
    $("aiJudgeButton").addEventListener("click", askJudge);
    $("aiPuzzleButton").addEventListener("click", generateAiPuzzle);
    $("hintButton").addEventListener("click", askForHint);
    $("aiHintButton").addEventListener("click", askAiHint);

    $("worldSvg").addEventListener("click", onSvgClick);
    // One document-level handler drives the player whether the drawing, a button or the body has
    // focus; the notebook keeps its own arrow keys and Enter (see onKeyDown).
    document.addEventListener("keydown", onKeyDown);

    showBrief(null);
    renderSuggestions();
    renderEvents();
  }

  function start() {
    try {
      wire();
      renderControls(null);
      loadAiSettings();
      boot();
    } catch (error) {
      try {
        showError("The page failed to start: " + (error && error.message ? error.message : String(error)), null);
      } catch (inner) {
        /* nothing left to do: never throw out of start() */
      }
    }
  }

  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", start);
  else start();
})();
