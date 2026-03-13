# ⚙️ proCLI

**proCLI** is a terminal-first toolkit for software projects.

It helps teams **validate environments, coordinate work, and automate project operations** directly from the command line.

The goal is simple:

> Turn project documentation and operational knowledge into executable tools.

Instead of relying on onboarding docs and tribal knowledge, projects can encode their requirements directly into a CLI.

## Quick Demo

Validate your development environment:

```bash
procli check
```

Start a Scrum poker session:

```bash
procli host-vote --story "JIRA-123 Fix login"
```

Join the session:

```bash
procli join localhost --session PO3BMI --name Alice
```

Example client output:

```
Session: PO3BMI
Story: JIRA-123 Fix login

Current clients:
 - Alice
 - Bob
```

## Project Status

proCLI is an **early-stage project**.

Current focus:

- building a solid architecture
- implementing collaboration tools
- enabling project diagnostics

The networking layer for Scrum poker is currently under active development.

Contributions and feedback are welcome.

## Vision

The long-term goal of proCLI is to become a **terminal-native project assistant**.

Instead of juggling multiple tools, developers should be able to run:

```
procli doctor
```

and immediately understand:

- if their environment is ready
- what tools are missing
- how to fix issues
- how to collaborate with their team

proCLI aims to bring project operations directly into the terminal.

## Good First Contributions

Areas where contributions would be helpful:

- voting mechanics for Scrum poker
- vote reveal logic
- Bubble Tea TUI for poker sessions
- project diagnostic checks
- documentation improvements

Contributions are welcome.

Typical workflow:

```bash
git checkout -b feature-name
git commit
git push
```

Open a pull request once your feature is ready.

## Why proCLI exists

Every software project has hidden operational rules:

* required tools
* environment variables
* tokens and credentials
* onboarding steps
* development workflows
* collaboration rituals

These rules often live in:

* READMEs
* internal wiki pages
* Slack threads
* CI scripts
* team knowledge

Developers spend time **discovering these rules instead of coding**.

**proCLI turns those rules into commands.**

Example:

```bash
procli check
```

Instead of reading documentation, you get immediate feedback:

```
✔ docker installed
✔ python version OK
✖ missing environment variable: CORPORATE_CLA
```

The same philosophy applies to **team collaboration**.

Why open a browser tool for Scrum poker when a terminal command could do it?

```bash
procli host-vote
```

---

## Experimental: Scrum Poker Sessions

proCLI includes an early prototype for **terminal-based Scrum poker sessions**.

Start a session:

```bash
procli host-vote --story "JIRA-123 Fix login"
```

Example output:

```
Starting session PO3BMI on :32896
```

Join a session:

```bash
procli join localhost --session PO3BMI --name Alice
```

Client output:

```
Session: PO3BMI
Story: JIRA-123 Fix login
Current clients:
 - Alice
 - Bob
```

Current capabilities:

* TCP session hosting
* session code validation
* client identity management
* automatic guest naming
* connection cleanup
* shared session context
* story broadcast

Voting mechanics will be added next.

---

# Architecture

proCLI is designed to grow without coupling features together.

```
cmd/    CLI commands
core/   domain logic
store/  persistence
net/    networking
ui/     terminal interfaces
```

Principles:

* networking is UI-agnostic
* domain logic is transport-independent
* persistence is isolated
* UI consumes domain state

This architecture allows features like diagnostics, collaboration tools, and automation to evolve independently.

---

# Installation

### Requirements

* Go 1.20+

### Build

```bash
git clone <repo>
cd procli
go build
```

### Install

```bash
go install
```

---

## Roadmap

proCLI is evolving into a **terminal-native project operations toolkit**.

Planned capabilities include:

### Scrum Poker

* vote casting
* vote reveal
* vote schemes (Fibonacci, T-shirt)
* remote relay server for distributed teams

### Project Diagnostics

Inspired by tools like:

```
flutter doctor
cargo doctor
```

Example:

```bash
procli doctor
```

Projects will be able to define their own environment diagnostics.

### Terminal UI

Using **Bubble Tea** to provide interactive interfaces for:

* Scrum poker
* project configuration
* diagnostics

### Plugin System

Projects will eventually be able to extend proCLI with custom checks and automation.

## Philosophy

proCLI embraces a **terminal-first approach** to development tooling.

The terminal is universal:

* works over SSH
* works inside containers
* works in remote environments
* works everywhere developers already work

Instead of replacing existing tools, proCLI aims to **compose them into a cohesive interface**.

## License

MIT License — see `LICENSE`.

