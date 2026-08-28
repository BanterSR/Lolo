---
name: lolo-game-control
description: Launch the installed official Lolo game client from a stopped state and perform a bounded scenario with Windows Computer Use after Iridium capture is active. Do not use for attaching to an already running game, packet analysis, terminal automation, or server code edits.
---

# Lolo Game Control

Use the available Windows Computer Use capability and read its complete `SKILL.md`, core guidance, and confirmation policy before any UI action.

## Preconditions

- Confirm the official game process is stopped. Selecting an already running game is not valid for a full official-data capture.
- Confirm a fresh Iridium session already reports `capturing` with an empty packet filter. Capture must begin before this skill launches the game.
- Reduce the requirement to a short action sequence with an observable end state.
- Use an exact user-provided executable path, a machine-local `LOLO_GAME_EXE` value, or an installed launcher returned by Computer Use. Never commit a machine-specific executable path.

If the game is already running, exit it cleanly and return control to `$lolo-iridium-capture` to create a fresh session. Do not continue with the existing process or describe its traffic as complete.

If authentication, CAPTCHA, a security prompt, or account selection appears, stop and hand that step to the user. Do not automate authentication dialogs or change anti-cheat, privacy, proxy, certificate, firewall, or other security settings.

## Window Control

List applications/windows and select exactly one returned game window. Never construct a guessed window handle. Activate it, observe current state, perform one input action, then refresh state before the next action. Do not reuse screenshot coordinates or accessibility indexes after any visual change.

Launch the game only after all preconditions pass. Keep the same capture active across startup, dispatch/login, server entry, and the requested feature. Perform only the scenario needed for the requested feature after startup completes. Mark these checkpoints in the work log:

1. Iridium capture confirmed active while no game process exists;
2. game process launched;
3. login/server entry completed;
4. immediately before the trigger action;
5. trigger action completed;
6. expected client result or error visible;
7. follow-up notices settled or the client exited.

Representational actions such as chat, friend requests, purchases, account changes, or other externally consequential actions follow the Computer Use confirmation policy. Ordinary local movement and menu navigation do not need extra confirmation.

## Handoff

Return a compact scenario record containing:

- window/app identity and client region/version when visible;
- exact actions and observed labels or values;
- timestamps for capture-ready, process launch, login/server entry, before-trigger, and settled checkpoints;
- whether the action succeeded, failed, or was blocked;
- confirmation that the same Iridium session covers the full flow from process launch.

Do not interpret packet semantics or edit server code in this skill.
