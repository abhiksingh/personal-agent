# Personal Agent macOS v2 Design System

Status: `v0.3`  
Last updated: `2026-03-05`  
Inspiration source: premium travel-assistant app screen language (not marketing site chrome).

## 0) Trust Console Thesis (Build Contract)

The product should feel like a trust instrument: calm, dense, and explicit about what the assistant received, decided, and executed.

Use these as non-negotiable build instructions:

1. Evidence first, chat second: default focus is auditable action history.
2. One dominant path: select instruction -> inspect proof -> approve/reject -> verify outcome.
3. Color is semantic only: neutral surfaces by default; color highlights urgency or primary action.
4. Progressive disclosure: summary in-line, deep trace behind explicit expansion.
5. Coherence over features: remove UI that duplicates status, intent, or action.
6. “Trust Pulse” owns top-level health summary:
   - `Waiting`
   - `At Risk`
   - `Automated Safely`
7. Replay defaults to `Needs Approval` to bias attention toward pending risk.

## 1) Design Intent

Build a trust-first operator UI with:

- High legibility first (clear hierarchy, low ambiguity).
- Glossy but restrained depth (glass surfaces, soft glows, subtle shadows).
- Status-forward color semantics (users understand state in <1s).
- Fast scanning in dense timelines without visual noise.

## 2) Core Principles

1. One dominant action per screen region.
2. Every state has a visible signifier: `Running`, `Needs Approval`, `Failed`, `Done`.
3. Cards float above layered atmospheric backgrounds, never flat white on white.
4. Use color to encode meaning, not decoration.
5. Dense data must still feel calm: strong structure + soft surfaces.
6. Accent blue is reserved for interaction focus (selection, primary action, active filter), not neutral container backgrounds.

## 3) Color Tokens

Use semantic tokens, not raw hex in views.

### 3.1 Base Neutrals

| Token | Hex | Use |
|---|---|---|
| `color.bg.canvas` | `#010A19` | Deep app background (dark sections) |
| `color.bg.canvas.alt` | `#030014` | Alternate deep background |
| `color.surface.primary` | `#18181B` | Primary dark card |
| `color.surface.secondary` | `#25262A` | Secondary card |
| `color.text.primary.dark` | `#FFFFFF` | Text on dark surfaces |
| `color.text.primary.light` | `#000000` | Text on light surfaces |
| `color.text.secondary.dark` | `#D1D5DB` | Supporting text on dark |
| `color.text.secondary.light` | `#4B5563` | Supporting text on light |

### 3.2 Semantic Accents

| Token | Hex | Use |
|---|---|---|
| `color.accent.info` | `#099CED` | Info, links, active route |
| `color.accent.info.deep` | `#00168D` | High-contrast info surfaces |
| `color.accent.promo` | `#5500DC` | Highlight glow, premium chips |
| `color.status.success` | `#08A85B` | Completed / healthy |
| `color.status.warning` | `#F7BE00` | Delay / caution |
| `color.status.danger` | `#E68670` | Failure / high risk |
| `color.status.neutral` | `#9CA3AF` | Unknown / queued |

### 3.3 Gradients (Background Only)

| Token | Value | Use |
|---|---|---|
| `gradient.night.base` | `#030014 -> #010A19` | Main dark canvas |
| `gradient.alert.warm` | `#4A0F14 -> #FF6A00` | Delay/alert panels |
| `gradient.info.cool` | `#0D1B6D -> #099CED` | Background atmosphere only |
| `gradient.twilight` | `#1A0A38 -> #5B3FD0` | Background atmosphere only |

## 4) Surface & Depth Tokens

### 4.1 Radius

- `radius.sm = 10`
- `radius.md = 12`
- `radius.lg = 16`
- `radius.xl = 20`
- `radius.pill = 999`

### 4.2 Border/Stroke

- `stroke.soft = 1px rgba(255,255,255,0.10)` on dark cards.
- `stroke.light = 1px rgba(0,0,0,0.08)` on light cards.

### 4.3 Shadow/Glow

- `shadow.card`: soft stacked shadow, low alpha.
- `shadow.float`: larger blur for elevated inspector panels.
- `glow.info`: outer glow with `color.accent.info` at low alpha.
- `glow.promo`: outer glow with `color.accent.promo` at low alpha.

### 4.4 Material

Use SwiftUI built-in materials for glossy controls:

- Top bar / command rail: `.ultraThinMaterial` + dark tint overlay.
- Floating chips: `.thinMaterial` + subtle inner stroke.
- Card surfaces: `.ultraThinMaterial` with low-opacity semantic tint overlays.
- Never exceed 2 layered blur surfaces in one viewport region.

## 5) Typography

### 5.1 Families

- Primary: `SF Pro Display` (headlines)
- Secondary/body: `SF Pro Text`
- Numeric/status dense fields: `SF Mono` optional for IDs/timestamps (Advanced mode only)

### 5.2 Scale

- `display.l = 56/60` (marketing/empty state only)
- `h1 = 40/44`
- `h2 = 32/36`
- `h3 = 24/30`
- `title = 20/26`
- `body = 16/24`
- `caption = 13/18`
- `micro = 11/14`

### 5.3 Rules

- Headline tracking: slight negative (`-0.3` to `-0.8`) only on large titles.
- Body text must keep `>= 4.5:1` contrast.
- Avoid all-caps except tiny status tags.

## 6) Component Patterns (App-Specific)

### 6.1 App Shell

- Background: `gradient.night.base` with subtle noise texture.
- Sidebar cards: glass-dark with `radius.lg`.
- Active nav item must use the same selection treatment as replay rows (shared stroke/tint/shadow hierarchy).

### 6.2 Activity Feed Rows

- Layout: `Source icon -> primary line -> secondary metadata -> state badge`.
- Each row gets one dominant status badge only.
- Hover: increase surface brightness by ~4% (not scale).
- Selection style must match sidebar selection style.

### 6.3 Replay Detail

- Top action rail fixed with one primary CTA.
- Group details into collapsible cards:
  - `Instruction`
  - `Reasoning Summary`
  - `Actions Taken`
  - `Evidence`
- Evidence cards use darker sub-surface and monospaced metadata in `Advanced`.

### 6.4 Approval Queue (Cross-Cutting)

- Separate visual lane from per-message detail.
- High risk rows: warm accent border + explicit confirmation affordance.
- Approve/Reject controls always persistent in detail footer.

### 6.5 Connectors & Models

- Show state chips inline: `Connected`, `Needs Setup`, `Degraded`.
- Setup buttons use solid fill; diagnostics use quiet style.
- Never mix destructive actions beside primary setup CTA.

## 7) Motion

- Default transition: `180ms` ease-out.
- State chip/color transition: `120ms`.
- Panel switch: `220ms` with slight fade + vertical offset (4-8pt).
- Avoid bounce on operational data surfaces.
- Respect `Reduce Motion`: remove transforms, keep opacity-only transitions.

## 8) Accessibility & Legibility Constraints

1. Contrast:
   - Text: `>= 4.5:1`
   - Large text: `>= 3:1`
2. Hit targets: minimum `44x24` for dense desktop controls, `44x44` for primary controls.
3. Never encode status by color alone; pair with icon/label.
4. Focus rings must be visible on dark and light backgrounds.
5. In `Simple` mode, hide low-value debug strings by default.

## 9) SwiftUI Tokenization Contract

Implement as centralized tokens before UI expansion:

- `DesignColor.swift` (`Color` extensions / asset names)
- `DesignTypography.swift` (text styles)
- `DesignRadius.swift`
- `DesignShadow.swift`
- `DesignGradient.swift`
- `DesignSpacing.swift`

No view should declare raw hex, ad hoc shadows, or one-off corner radii.

## 10) Do/Don’t

### Do

- Keep backgrounds atmospheric and surfaces structured.
- Use semantic status chips everywhere operational state appears.
- Use one bright accent at a time per panel.
- Prefer built-in control styles first (`.bordered`, `.borderedProminent`, segmented picker, `DisclosureGroup`) before adding custom variants.

### Don’t

- Flat gray utility UI with no depth.
- Multiple competing accent colors in one card.
- Blue-toned container fills on neutral informational cards.
- Expose advanced internals in default `Simple` paths.
- Hide actionable failures behind passive copy.

## 11) Change Management

When updating this system:

1. Add/modify token, do not fork local style.
2. Document rationale + before/after screenshot in PR notes.
3. Update this file version/date.
4. Validate with manual checks:
   - scan speed in Activity feed
   - status comprehension in Approval queue
   - setup clarity in Connectors/Models

## 12) Reference Capture (for future recalibration)

Primary references sampled on `2026-03-05` from modern travel-assistant app visuals:

- Hero app screen (light, map + notification chips)
- Dark delay prediction card stack (violet glow, warm alert chips)
- Warm/cool split card pair (orange alert, green operational panel)
- Friends/roster app screen (light list readability)
