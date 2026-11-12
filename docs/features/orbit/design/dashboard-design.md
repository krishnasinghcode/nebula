# Orbit Dashboard — UI Design System

> **Status: Implemented (v0.1.0).** This document records the dashboard's visual design so
> future components stay consistent without re-deriving it. The dashboard is a **developer
> console**, not a marketing product: dense, technical, calm — closer to the AWS Console
> aesthetic than to a startup landing page.

Related: [orbit-usage.md](../../../guides/orbit-usage.md) (user-facing behaviour),
[implementation-plan.md](implementation-plan.md) (phase 9 record).

## 1. Design principles

1. **Console, not product site.** Flat panels, hairline borders, no gradients, no rounded
   blobs, no illustrations, no "welcome back" copy. Heavy software should *look* like
   heavy software.
2. **Density over air.** Tight paddings, compact table rows. Screens are for operators.
3. **One accent, used only for actions.** The accent color appears on primary buttons,
   links and active filters — never for decoration. State colors (ok/warn/bad) appear
   only in status indicators and toasts.
4. **Monospace for technical values.** Instance IDs, IPs, ports, types, fingerprints,
   SSH commands, user data are monospace. Names and labels are sans.
5. **Uppercase micro-labels for structure.** Section titles and table headers are small,
   uppercase, letter-spaced, muted — like console chrome.
6. **Motion is functional.** 100–150ms color transitions only. No bounce, no parallax.

## 2. Theming

Theming is CSS-variable based (`orbit/dashboard/src/index.css`). Components use semantic
Tailwind color utilities mapped in `@theme inline` — never raw palette colors (`zinc-*`,
`blue-*`) except inside `index.css` itself.

| Utility | Meaning |
|---|---|
| `bg-bg` | App background |
| `bg-panel` | Card/section surface |
| `bg-hover` | Row hover / quiet button hover |
| `border-line` | All borders (hairlines) |
| `text-ink` | Primary text |
| `text-dim` | Secondary text, labels, empty states |
| `text-brand` / `bg-brand` | Accent (actions only) |
| `text-ok` `text-warn` `text-bad` | State colors (status, toasts, destructive) |

- Default theme: **light** (soft green-tinted: `#f2f7f3` bg, white panels, deep-green
  accent `#1a7f4b`). Dark theme: graphite (`#09090b` bg, `#111114` panels, blue accent).
- The active theme is `data-theme` on `<html>`, set in `main.tsx` before first paint,
  persisted in `localStorage["orbit-theme"]`. Toggle: `ThemeToggle` in the top bar.
- **Rule:** any new color need becomes a token in both palettes, not a one-off class.

## 3. Shared primitives (`index.css`)

| Class | Use |
|---|---|
| `.console-input` | Every input/select/textarea |
| `.btn-primary` | The single primary action of a view |
| `.btn-danger` | Destructive action — **outline** red, not filled (danger is typographic) |
| `.th-label` | Table headers: 11px, uppercase, tracking-wider, dim |

Quiet secondary buttons: `border border-line hover:bg-hover` (Stop/Reboot/Add/Prev/Next).

## 4. Layout patterns

- **Top bar (sticky):** wordmark `NEBULA ORBIT` (uppercase, tracking-widest), capacity
  meters, theme toggle, Change token. Full-width strip on `bg-panel`.
- **Panels:** sections are `rounded-lg border border-line bg-panel` — one border, no shadow.
- **Table:** row = 1px bottom border, hover `bg-hover`, click → detail. Empty state is a
  quiet centered line ("No instances. Create one below."), never a big illustration.
- **Capacity:** thin segmented meters (vCPU / Mem / Inst) with mono `used/max`; turns red
  at ≥90%. Lives in the top bar, with an ⓘ explanation.

## 5. Interaction patterns (do not reinvent)

| Pattern | Component | Rules |
|---|---|---|
| **ⓘ section help** | `components/Info.tsx` | One ⓘ per section title (small, dim; brightens when open). Click opens a popover: **one brief line** + "See more ▾" revealing bulleted detail (`InfoList`). Closes on outside click / Escape. Click on ⓘ must not trigger parent labels (`preventDefault`). |
| **Toasts** | `components/Toast.tsx` | Global bus: `toast(msg)` (error) / `toast(msg, "success")`. Bottom-right stack, auto-dismiss ~4.5s. API errors AND confirmations surface here. |
| **State display** | `components/StateBadge.tsx` | Colored dot + monospace lowercase state text. Dot colors: running ok, provisioning/rebooting brand, stopping/terminating warn, stopped dim, failed bad, terminated hollow. |
| **Filters** | `InstanceTable` | Mono chips; `active` (default) = everything except terminated. |
| **Pagination** | `InstanceTable` | 5 rows/page, Prev/Next + `n / m` indicator, only rendered when needed; resets on filter change. |
| **Confirmations** | native `confirm()` | On irreversible actions: terminate, key delete, change token. Styled in-app modal may replace this later. |

## 6. Content rules

- Buttons and states in lowercase/normal case; section titles uppercase.
- Technical values copied from the API are shown as-is (no reformatting), with `—` for empty.
- Help text explains *what it is and its rules* (uniqueness, shebang, key origin), not
  marketing. The trust-model wording ("containers sharing my host kernel") is security
  documentation — do not soften it.
- Never describe instances as VMs, and never claim hypervisor-level isolation.

## 7. Testing conventions

Behavior lives in `app.test.tsx` (Vitest + Testing Library) via `data-testid`:
`row-<name>`, `filter-<state>`, `page-prev/next`, `page-indicator`, `create-*`, `key-*`,
`trust-ack`, `info-button`, `toast-<kind>`, `theme-toggle`, `capacity-summary`,
`change-token`. New UI features add tests for behavior, not styling.
