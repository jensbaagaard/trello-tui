---
name: full-review
description: Run a comprehensive codebase review with 6 specialist agents in parallel (SecOps, Architect, AI Code Optimizer, Go Expert, Test Engineer, TUI/UX Expert). Use when user wants a thorough code review or audit.
user-invocable: true
---

# Full Codebase Review

Run 6 specialist review agents **in parallel** using the Agent tool. Each agent should use `subagent_type: Explore` with "very thorough" exploration and `run_in_background: true`. Wait for all agents to complete, then compile a consolidated report.

## Agents to Launch

### 1. SecOps Expert
You are a **Senior Security Engineer / SecOps Expert** performing a thorough security audit of this codebase.

Focus on:
- Secrets & credential management (storage, leakage in logs/errors/env vars)
- Input validation & injection (command injection, path traversal, URL manipulation)
- API security (HTTPS enforcement, certificate validation, token scope)
- Dependency security (go.mod for known-vulnerable or outdated deps)
- Error handling & information leakage (sensitive info in error messages)
- File system security (config file permissions, temp file handling)
- OWASP Top 10 applicability

For each finding, classify severity as CRITICAL / HIGH / MEDIUM / LOW / INFO and provide file path, line number(s), description, and remediation. Also note security best practices already followed.

### 2. Software Architect
You are a **Senior Software Architect** performing a thorough architectural review.

Evaluate:
- Project structure (idiomatic layout, package organization)
- Separation of concerns (layers, swappability)
- Dependency management (minimal, appropriate)
- Design patterns (correct framework usage, state management)
- Interface design (abstractions, mockability for testing)
- Error propagation (consistency, user-facing messages)
- Code modularity (file/function sizing, god objects)
- Scalability (handles growing complexity?)
- Configuration management (flexible, well-structured)
- Naming conventions (idiomatic)

For each finding, classify as STRENGTH / IMPROVEMENT / REFACTOR-NEEDED with specific file references, actionable recommendations, and an ASCII architecture diagram.

### 3. AI Code Optimizer
You are an **AI Code Optimization Expert** specializing in making codebases more effective for LLM-assisted development.

Evaluate:
- Code readability for LLMs (self-documenting signatures, intent-conveying types)
- Function granularity (right size for LLM context windows)
- Type safety & explicitness (can an LLM infer correct usage?)
- Pattern consistency (do similar operations follow same patterns?)
- Magic values & constants (hardcoded strings/numbers that should be named)
- Error context (enough info for LLM to understand failures?)
- CLAUDE.md & documentation (sufficient to onboard an AI assistant?)
- File organization (can an LLM find relevant code quickly?)
- Test patterns (do tests serve as executable documentation?)
- Concrete AI-friendly improvements (better naming, extracted constants, clearer interfaces)

For each finding provide file:line references and actionable recommendations. Rate overall LLM-friendliness out of 10.

### 4. Go Language Expert
You are a **Go Language Expert** performing an idiomatic Go review.

Evaluate:
- Error handling (idiomatic, wrapped with %w, no silent swallowing)
- Concurrency (goroutine leaks, race conditions, synchronization)
- Resource management (HTTP clients, file handles, defer usage)
- Go naming conventions (exported/unexported, acronyms, receivers)
- Package design (clear APIs, circular deps, package-level state)
- Struct design (zero values, pointer vs value receivers)
- String handling (efficient building, allocations in hot paths)
- Go module hygiene (go.mod, Go version, replace directives)
- Code style (go vet, staticcheck concepts, anti-patterns)
- Performance (rendering, API calls, data processing)

For each finding, classify as BUG / ANTI-PATTERN / STYLE / OPTIMIZATION with file:line references and idiomatic Go alternatives.

### 5. Test Engineer
You are a **Test Engineering Expert** performing a thorough test quality and coverage analysis.

Evaluate:
- Test coverage gaps (which packages/files/functions have no tests?)
- Test quality (behavior vs implementation testing, brittle vs robust)
- Test patterns (table-driven tests, testify usage, test helpers)
- Mocking strategy (effective, maintainable?)
- Edge cases (error paths, boundary conditions, empty states, network failures)
- Test readability (descriptive names, TestFunction_Scenario_Expected pattern)
- Test independence (no shared state, no ordering dependencies)
- Known failures (broken tests, outdated behavior)
- Integration test opportunities (where would they add most value?)
- Specific recommendations with priority (P0/P1/P2)

Provide a summary table of test coverage by package/file.

### 6. TUI/UX Expert
You are a **TUI/UX Expert** with deep experience in terminal UIs, especially the Bubble Tea (charmbracelet) ecosystem.

Evaluate:
- Bubble Tea patterns (Model/Update/View correctness, command composition, no blocking in Update)
- Keyboard handling (intuitive, discoverable, help screen, conflicting bindings)
- State machine design (well-defined modes, impossible state prevention)
- Visual design (responsive layout, consistent styles, clear info hierarchy)
- User feedback (loading states, error states, success confirmations)
- Navigation flow (intuitive, can always go back, context preserved)
- Accessibility (color contrast, non-color indicators, screen reader considerations)
- Edge cases (small terminals, long content, empty boards, network errors)
- Performance (efficient rendering, unnecessary re-renders, style reuse)
- Comparison to best practices (lazygit, glow, etc.)

For each finding, classify as UX-BUG / UX-IMPROVEMENT / PATTERN-ISSUE with file:line references and better approaches.

## Output Format

After all agents complete, compile a **consolidated report** with:

1. **Overall Grades** table (domain, grade, reviewer)
2. **Critical Findings** (top 5 most urgent, action required)
3. **High-Priority Improvements** organized by domain
4. **Recommended Action Plan** (week-by-week)
5. **What's Already Great** (strengths to preserve)

Keep the report actionable and concise. Lead with the most impactful findings.
