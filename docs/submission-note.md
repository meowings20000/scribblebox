# Scribblebox — one-page technical note

Task 2, section 5 (research source, what I built, what the model drafted, one rejected suggestion, one next step). Project: `github.com/meowings20000/scribblebox`.

## Research source

**Slamecka, N. J., & Graf, P. (1978). The generation effect: Delineation of a phenomenon. *Journal of Experimental Psychology: Human Learning and Memory, 4*(6), 592–604.** Words a learner *generates* are remembered better than the same words read. Kapur, M. (2008), *Productive failure*, *Cognition and Instruction, 26*(3), 379–424, supplies the other half: an attempt that fails first, with immediate feedback, prepares better learning than an explanation given up front.

**Mapping, in one line:** the learner must *produce* the name of the object they need and type it from memory — never pick it from a list — and the world immediately answers with what that object can really do, so every attempt is a generated attempt that receives feedback.

**What would count as the idea failing:** if the game told the learner the answer, or if a wrong choice behaved the same as a right one, both the generation and the feedback would be gone; the honest test is whether a learner can still name the objects afterwards without the app.

## What I built

Scribblebox: type an object's name into the notebook and it appears in a 2D world, where it behaves like itself — a ladder can be climbed, a match starts a fire, anything cold freezes water into a bridge, dynamite opens a locked chest. One activity, four puzzles, one loop: prompt → write → see the consequence → try again. 284 nouns carry properties the engine acts on, modifiers change an object before it appears ("big flaming ladder", "metal rope"), and a word the dictionary has never heard of is improvised rather than refused, so a learner is never blocked. The whole world is inline SVG the page draws itself (no images, no fonts, no libraries) and runs offline. Verified by 139 Go tests, 252 frontend checks and 20 browser specs, two of them against the live Docker stack; a 5 minute 40 second recorded session is in `docs/scribblebox-demo.mp4`. The optional AI panel takes the reviewer's own endpoint and key: the key is kept in the browser, sent only to make the request, and never stored or logged on the server.

**Interaction path:** typed input, no ASR/TTS, so the brief's voice example does not apply. Prompt, learner action, feedback and retry all happen in the same session, in one screen.

## What the model drafted

The 284-noun dictionary with its property tags, the world engine's ten physics rules and the entire SVG frontend were drafted by three coding agents running in parallel, each given a pinned interface first: the object struct and the closed tag vocabulary, the world API with the rule order, the HTTP contract and the DOM ids. I wrote the AI layer and the HTTP layer myself, with model help on the prompt wording, and I pinned every interface before delegating. What I then changed by hand is listed in `docs/TECHNICAL_NOTE.md` — it includes the four-choice promise below, a CORS allowlist that no longer disagrees with the published port, and a repair round that hands the engine's own error back to the model when a generated puzzle does not fit.

## One suggestion I rejected, and why

The first design checked the four-choice hint **when it was handed out**: simulate the four options once, show them, and trust them from then on. I rejected it because the player keeps playing in between — the character walks, stacks grow, fire spreads — so an option that was verified can stop working exactly when the learner relies on it, which is when a promise like "one of these works" does its damage. The press now goes to the server, which re-derives a spot that is verified for the world *as it stands*, settles the world, and only then answers. A test deliberately changes the world between the two moments and requires exactly one of the four to still win.

## One next step

Measure it with learners instead of adding content: a short productive-vocabulary pre/post — name twenty objects before and after fifteen minutes of play, with no app present in the test. If the words a learner generated and watched fail do not transfer, then the mechanism is not doing the work and the interaction needs redesigning, not another four puzzles.
