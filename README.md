<svg height="24" width="117" xmlns="http://www.w3.org/2000/svg">
    <rect width="29" height="24" fill="#000000" />
    <rect x="29" width="88" height="24" fill="#bb400c" />
    <text text-anchor="middle" font-weight="bold" font-size="15" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" fill="#ffffff" x="15" y="50%" dy=".35em">⚙️</text>
    <text text-anchor="middle" font-size="19" font-family="Verdana,Geneva,DejaVu Sans,sans-serif" fill="#ffffff" x="73" y="50%" dy=".35em">proCLEE</text>
</svg>

---

**clee** or **proCLEE** is a terminal‑first **software project assistant** built in Go using **Bubble Tea**.

It is designed to grow into a **modular, extensible TUI toolbox** for software teams: Scrum facilitation, project health checks, diagnostics ("project doctors"), and custom workflows — all from the terminal.

The current codebase contains only the **first foundational brick**: a local, deterministic random chooser. Everything else will be layered on **only after full understanding and ownership** of each step.

---

## Vision

`clee` aims to become:

* A **CLI/TUI assistant** for running software projects
* Extensible via **modules / plugins** (conceptually, not dynamically loaded yet)
* Opinionated where it helps, customizable where it matters

Think:

* **Scrum poker, standups, retros** (later)
* **Project diagnostics** similar to `flutter doctor`, but:

  * configurable
  * project‑specific
  * language / stack agnostic

The terminal is the primary UI.

---

## Current MVP (first brick)

The current implementation is intentionally small. It exists to:

* Establish the TUI foundation (Bubble Tea)
* Define UX patterns we will reuse later
* Ensure correctness, determinism, and understanding

### What the MVP does

* Manage a list of participants stored in a **simple text file**
* Enable / disable participants via **checkboxes**
* Add, edit, and delete names
* Start a **random selection** with a visual animation
* Pick a **truly random winner** (winner chosen first, animation is cosmetic)
* Display a dedicated **Winner 🎉 screen**

Everything runs **locally** in the terminal.

---

## Screens

1. **Participant list**

   * Arrow keys to navigate
   * Checkbox per participant (included / excluded)

2. **Edit screen**

   * Add a new name
   * Rename an existing one

3. **Selection animation**

   * Cursor cycles through enabled participants
   * Slows down and lands on the pre‑selected winner

4. **Winner screen**

   * Displays: `winner 🎉 {name}`
   * Return to list to run again

---

## Key bindings

| Key            | Action                    |
| -------------- | ------------------------- |
| ↑ / ↓ or k / j | Move cursor               |
| Space          | Toggle participant on/off |
| a              | Add participant           |
| e              | Edit participant          |
| d / Backspace  | Delete participant        |
| Enter          | Start random selection    |
| q / Ctrl+C     | Quit                      |

On the **Winner screen**:

* `Enter` or `Esc` returns to the list

---

## Data storage

Participants are stored in a plain text file:

```
~/.config/clee/participants.tsv
```

Format:

```
<enabled>\t<name>
```

Example:

```
1	Alice
0	Bob
1	Charlie
```

* `1` = enabled
* `0` = disabled

This format is intentionally simple and editable by hand.

---

## Build & run

Requirements:

* Go 1.20+

```bash
go mod tidy
go build -o clee .
./clee
```

Or install into your Go bin directory:

```bash
go install .
```

---

## Design principles

These principles apply to the entire project:

* **Foundation first** — no features without understanding
* **Winner / result first, animation second** — logic must be correct
* **Small composable bricks** — each feature stands alone
* **Terminal as a first‑class UI**
* **No premature complexity**

Feature growth is gated intentionally.

---

## Roadmap (pinned, not implemented yet)

The following are **explicitly out of scope for the current MVP**:

* Scrum poker (multiple voting schemes)
* TCP / socket networking
* Session discovery & codes
* Multi‑client coordination
* Project diagnostics ("project doctors")
* Plugin / module registry
* WebAssembly / browser spectator UI
* Reports / exports

They will only be implemented after explicit confirmation.

---

## Status

**Status:** foundational TUI brick complete and understood.

This repository will evolve into a broader **software project assistant**, step by step, without skipping understanding.
