# Demo script (about four minutes)

Everything below is a real flow on the running stack (`docker compose up -d --build`,
then <http://localhost:3003>). Nothing here needs AI configured; the AI part at the end is optional.

## 1. The notebook writes the world (40 s)

1. Open the game. Point at the four puzzles and the line: **"Write an object's name in the notebook and
   it appears in the world."**
2. Pick **The star in the tree**.
3. Type `ladder`, press **Write it**. The ladder lands above you, falls, and rests on the ground.
4. Type `big flaming torch`. Note the notebook says the book knew the word, and the object list shows
   its tags: `fire, light-source, flammable, tool`.
5. Type `zorble`. Say the line that matters: **the game never refuses a word** — it improvises a stable
   object and tells you it did.

## 2. Objects behave like themselves (40 s)

6. Press **Advance time** a few times: the torch burns, whatever is flammable near it turns to ash, and
   the event log narrates it ("the scaffolding crushes the cake" is a favourite).
7. Walk right with **Right** and jump with **Jump**; the little character climbs the ladder and the goal
   banner updates as you get closer.

## 3. Four choices, one of which really works (60 s)

8. Press **Give me four choices**. Four objects appear, each drawn as the icon it will have in the
   world. Say: **exactly one of these solves the puzzle, and the game proved it by simulating all four
   before showing them to you — the right one is not marked anywhere.**
9. Pick a wrong one on purpose: the line says honestly *"did not reach the goal — try another of the
   four"*.
10. Pick the right one: it is placed at a spot the engine verified **for the world as it stands**, the
    world settles, and the banner turns to **Solved!** with the line *"… worked and the goal is met"*.
11. If you like, press the hint button after piling things up in one place: the engine moves the spot
    rather than refusing, and if nothing can be verified it says so and points at **Reset** instead of
    guessing.

## 4. The other three puzzles (30 s)

12. **Cross the river** — `bridge` (or `boat`, or `ice` on the water, or a jetpack).
13. **Light the three candles** — `match` next to each candle.
14. **Open the locked chest** — `key`. Then `dynamite` on a fresh run to show a very different route.

## 5. Optional: your own AI endpoint (60 s)

15. Open **AI features (optional)**, tick *Use AI features*, enter your base URL, key and model
    (`http://host.docker.internal:3000/v1` if your gateway runs on this machine), press **Save AI
    settings**, then **Test connection**. It reports the model and the latency, or tells you exactly
    what the endpoint said (refused key, timeout, unreachable).
16. **Ask the AI judge**: *"would a burning rope cut the branch the star sits on?"* The verdict card
    shows approved/denied, the reason and the model that answered; an approval is recorded by the engine
    and the goal state says a judge accepted it.
17. **Invent a puzzle with AI**: theme `a lighthouse in a storm`, goal kind `star`, press the button.
    The generated puzzle opens badged as AI-made — and if the model sends something the engine cannot
    referee (too big, goal too far, a missing star) the UI says *"the AI's puzzle was rejected: …"*
    instead of playing something broken.

## 6. Close (20 s)

18. Point at the footer: **no images, no fonts, no libraries — the world is inline SVG the page draws
    itself.** Then the tests: `go test -race ./...` (136), `npm test` (252 checks),
    `npm run test:e2e` (18 browser specs, two of them against this very stack).
