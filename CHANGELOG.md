# Changelog

All notable changes to gaem are documented here. This file follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) conventions and the project uses [Semantic Versioning](https://semver.org/).

> Releases prior to v0.1.4 (v0.1.0–v0.1.3) predate this changelog.
> Their commit-level release notes live on the [Releases page](https://github.com/klp2/gaem-releases/releases).

## [v0.1.5] - 2026-05-03

### Added

- Three new patron shrines — Pale Sister, Tongueless Saint, Antlered Huntress — joining the existing six. Every shrine now samples 3 of 6 class-affinity-weighted patrons from the full 9-patron roster, so two shrines on the same floor can offer different choices.
- Codex Patrons tab shows a per-class affinity hint, surfacing which patrons your class is biased toward when shrines roll their 3-of-6 sample.

### Fixed

- Shrine overlay descriptions no longer truncate mid-word. Box width grew from 60 to 72 columns, and the truncation routine is now rune-aware so multi-byte glyphs (such as the runic block) render an ellipsis cleanly.

## [v0.1.4] - 2026-05-03

### Added

- Press `D` to drop items from your inventory. Bag items drop instantly; equipped gear (Weapon / Armor / Ring) prompts to confirm before unequipping.
- When you pick up an item with a full bag, you're prompted to swap something in your inventory for the new item — completes in a single turn.
- Scrollable message log (`m` key) surfacing the full 80-message buffer in a bottom-anchored modal. PgUp/PgDn/↑↓/jk/Home/End scroll bindings; freshness gradient (gold-bold newest → silver/dim oldest).

### Changed

- Cursed Vigor's omen regen now surfaces an inline message ("Cursed Vigor mends you (+1 HP).") on each healing tick, so the boon's benefit is visible against the bane's starvation drain. Silent at full HP to avoid log spam.

### Fixed

- Multi-item tiles now render `*` as the stack glyph, and the examine cursor lists every item on a tile (previously the render and examine paths each stopped at the first item, hiding stacks from view).

[v0.1.5]: https://github.com/klp2/gaem-releases/releases/tag/v0.1.5
[v0.1.4]: https://github.com/klp2/gaem-releases/releases/tag/v0.1.4
