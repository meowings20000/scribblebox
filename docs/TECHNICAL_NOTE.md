# Technical note

How Scribblebox is put together, and which decisions I made **by hand** rather than delegating.

## Shape of the thing

```
browser (index.html + app.js, vanilla)
   │  same-origin /api/... (Nginx proxies inside Docker)
   ▼
backend/api      HTTP: worlds, hints, AI proxy, CORS, bounds, JSON strictness
backend/world    grid + terrain + rules + player + four puzzles + hint verification
backend/lexicon  284-noun dictionary, 51 modifiers, improvised words, autocomplete
backend/ai       OpenAI-compatible client, judge, puzzle generation, hint suggestions
```

The engine is deterministic and rule-based on purpose. Nothing in the game needs a model, a key or a
network: the AI package is an optional layer that can only *propose* things, and the engine decides
whether a proposal is acceptable.

## Where the work came from

I built the layers that define behaviour and safety myself, and handed three well-bounded slices to
parallel subagents:

| Slice | Who | What I pinned before handing it over |
|---|---|---|
| `backend/lexicon` | subagent | The exact `Object` struct, the closed tag vocabulary, the shape list, the modifier semantics |
| `backend/world` | subagent | The exact `World` API, the ten physics rules and their order, the goal kinds, the easiness limits |
| `frontend` | subagent | The whole HTTP contract (routes, JSON shapes, ids), the renderer rules, the style guide |
| `backend/ai`, `backend/api` | me | — |
| Integration, tests, docs, verification | me | — |

Pinning interfaces first is what made parallel work possible: the three slices only met at the
contract, and every interface the subagents had to honour was written down (including the tag strings,
because the world engine reads them).

## Things I changed by hand after seeing the result

Subagents produce plausible work; these are the places where "plausible" was not good enough.

1. **A test of mine hung the whole suite.** My timeout test used a handler that blocks forever, so
   `httptest.Server.Close()` waited for it and `go test ./...` never returned. Fixed by having the test
   handler sleep past the client's timeout and then return.
2. **The whole reason the hint needed a second endpoint.** "One of these four works" is verified when
   the choices are *offered*, but the player keeps playing: the walker moves, stacks grow, and a spot
   that was good a minute ago is full. So a press goes through
   `POST /api/worlds/{id}/hint/choose`, which re-derives a verified spot **at that moment** and settles
   the world (40 ticks, the same window the verification uses) before answering. The promise is now
   true when the player presses, not just when the panel was drawn.
3. **The hint spot moved when it had to.** `Hint()` originally refused as soon as the authored spot was
   crowded — honest, but it takes a child's help away. It now tries the authored spot and then a
   widening ring of nearby cells, and only refuses when no spot can be verified at all.
4. **A line that lied after a win.** Choosing an option often wins a tick or two later (the player
   climbs). The page was left saying "did not reach the goal" under a solved banner. The choice is now
   remembered and the line is amended as soon as the goal is met — and two browser specs pin both
   halves of that.
5. **Dead options left on screen.** When the engine cannot stand behind four choices it says so; the
   old options must not stay clickable. The panel now clears them and keeps the reason visible.
6. **The choices all looked the same.** The hint previews were generic crates because `/api/words`
   returned names only. It now returns the exact shape and colour of a word it knows, so each choice is
   drawn as the icon it will become.
7. **CORS broke when I moved the ports.** The frontend went to 3003 while the backend's allowlist still
   said 3001, so the browser's own same-origin requests were refused with 403 and the game never
   loaded. Fixed — and guarded: `TestDefaultsAllowThePublishedFrontendPort` reads
   `docker-compose.yml` and fails if the published frontend port is missing from the allowlist.
8. **A stale footer.** The page still claimed it drew everything "on this page's own canvas" after the
   renderer moved to inline SVG.

## What "exactly one is correct" means here

It is not a label on a button; it is a property the engine proves:

```go
// every authored solution is dry-run in a copy of the world
for _, solution := range w.puzzle.Solutions { solved, _ := w.trialLocked(solution.Phrases[0].Phrase, placement); … }
// every decoy must provably fail, or it is not offered
// and the finished set is re-checked: exactly one option may solve it
if solving != 1 { return HintSet{}, fmt.Errorf("the hint would be dishonest (%d of the options solve it)", solving) }
```

The client never receives which option is right: the options are plain strings, shuffled into a slot
derived from the world's id and state, and a test walks the JSON to prove no field named `correct`,
`solution` or `answer` exists anywhere in it.

## Sandboxing the AI

Three rules keep the model on a leash:

1. **It only proposes.** Puzzle specs, hint candidates and verdicts all go through the engine.
2. **The engine can refuse.** `world.NewFromSpec` validates a generated puzzle structurally *and* for
   difficulty; a rejection is reported to the player as "the AI's puzzle was rejected: …" and nothing is
   created.
3. **Credentials are per request.** They arrive from the player's browser, are used for that one call,
   and are never stored or logged. A test asserts the key never appears in a response body or an error
   message, including on the 401 path.

## Numbers

- Go: **136 tests** across five packages, green under `-race`; ~12,400 lines including tests.
- Frontend: **252 static checks** plus **18 Playwright specs** (2 live against Docker, 16 mocked).
- Dictionary: **284 nouns**, **51 modifiers**, 20 shapes, 35 tags.
- Puzzles: **4**, with **32 authored solutions** between them — every one simulated in a test.

## Known weak points

- Improvised objects are generic: the fallback gives them a stable shape and the modifiers you used,
  not powers the word implies.
- The player's walker is a simple climber: it walks and climbs towards the goal, and the puzzle layouts
  are kept simple enough (that is why they are 20×10) for it to be reliable.
- A determined player can make a world that can no longer be verified for a hint (by piling things up);
  the engine says so and points at Reset rather than guessing.
- The AI judge's verdicts are not re-checked by the engine against the world (they cannot always be);
  they are recorded as a judgement, and the goal state says so.
