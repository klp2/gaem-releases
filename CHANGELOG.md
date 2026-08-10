# Changelog

All notable changes to gaem are documented here. This file follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) conventions and the project uses [Semantic Versioning](https://semver.org/).

> Releases prior to v0.1.4 (v0.1.0–v0.1.3) predate this changelog.
> Their commit-level release notes live on the [Releases page](https://github.com/klp2/gaem-releases/releases).

## [v0.15.0] - 2026-08-10

### Added

- Picky looters now get a picker.

### Changed

- Frozen, Slowed, and Shocked enemies now visibly show their active status and obey the status effect's movement, attack, and ability restrictions.
- Examine mode explains enemy status effects in detail.
- Resuming a run now uses `[R]`, leaving `[C]` available for Grimoire Fortification.
- Nemesis enemies now retain their visible Enraged identity and correctly activate their class affixes.

### Fixed

- Exploding enemies now detonate consistently when killed by status effects, terrain, traps, or chained explosions.
- Explosion chains now resolve in a stable order, and all explosion kills grant the rewards and progress they should.
- Saves in a surprising state are less likely to break things.
- Capstone labels no longer render repeatedly in the Codex.
- Summoned enemies retain their combat statuses after resuming a saved run.

### Compatibility

- A run saved by this version cannot be resumed by an older build. Older builds back up the newer save and start fresh; the current build continues to migrate older saves forward.

## [v0.14.0] - 2026-08-09

### Added
- Items sold to the merchant now have their own tab in the merchant UI, and are not destroyed if you reroll the merchant's inventory.
- Examine mode gets a drawer: details appear in a panel below the map instead of spamming the message log.
- The messages pane [m] now has a lore panel, and there is more lore to find.
- Sealed crates and wish fountains list what they might pay before you spend.
- New background: the Cannibal.
- New enemies with new abilities to discover.
- Leveling up now says what its full restore actually paid: "Restored to full. (+38 HP, +19 MP)".

### Changed
- The rune pouch gains a cursor. Discarding is now a drop — [D] lands the fragment at your feet, at the cost of a turn, rather than destroying it.
- New runs carry their starting rations in the consumable pouch, not the gear bag. A run saved before this update keeps its bagged rations.
- The in-run controls hint fits every terminal width, and always shows the help key.
- Dungeon denizens have strong preferences on where to make their home.
- Player and mob resistance mechanics improved.
- More item types have a unique color in your inventory.
- R on the character screen no longer re-rolls the omen offer — the offer is part of the seed's challenge.
- Committed coin throws at a fountain spend a turn.
- The consumable pouch sorts in the same order as the bag.
- Worlds generate differently from seeds in this version.
- A save written by this version cannot be read by an older build — and that now includes your profile, not just the run in progress. An older build backs the file up and starts fresh, so downgrading looks like total progress loss.

### Fixed
- Enemies killed by an explosion's splash now grant XP, gold, loot, essence, bestiary and contract credit like any other kill.
- Kills from poison, burning, and lava count toward the run's kill totals. Resuming an active run from an older save tops the totals up.
- Loot can no longer hide the stairs or a vault's Seal pedestal — generated loot avoids them, and anything dropped there keeps the landmark visible.
- Character creation previews crit, dodge and attack with the run's own formulas — a Scavenger's Eye Mage could be shown a point less attack than the run actually gave.
- The Mastery ladder lines up in columns instead of stair-stepping.
- The Codex reserves its scroll gutter, so text no longer runs under the bar.
- Three achievement reward descriptions matched neither their code nor their effect.
- A failed World delete no longer leaves a ghost entry that reads as total progress loss on the next launch; a delete that changed nothing is the only thing now reported as "Delete failed".
- Quick-quaffing with no potion in your consumables pouch now says so, and names how many are sitting in your bag.
- An unrecognized command-line option is now an error instead of a silent terminal launch, and `--windowed` works as well as `-windowed`.
- Caps Lock no longer kills your letter commands in silence: after a few dead presses the game says "Nothing responds. Caps Lock?".
- The help screen's forge line taught the old controls: it now shows the arrows, Enter, and i/r as the actions they are.
- A lore scroll no longer vanishes on any key. Enter, Space or Escape closes it; anything else leaves it up.
- The mutations screen names the whole of what a sworn skull does: each one adds a Relic Shard to the purse of a run you win, not just the essence tithe.

## [v0.13.1] - 2026-08-03

### Fixed
- Starting a new character no longer carries anything over from a run you quit to the menu.
- Your once-per-run oath reroll refreshes with each new run again.
- The Worlds list is written more safely, so an interrupted save can't leave it damaged.
- The help screen scrolls, so every row is reachable — and it now covers the merchant's reroll and services keys.

## [v0.13.0] - 2026-08-02

### Added
- The Preserving Draught — holds hunger off for a while. You can't eat while it does.
- You gain a neck slot!
- The Fasting Amulet.
- The run summary breaks Damage Endured into your three heaviest sources, with the rest folded into one row.
- Name your character at creation, up to 12 characters. Leave it blank and the game rolls one for you.
- Settings gains a Fonts & Glyphs screen. If rune sigils show as boxes, press [I] to install gaem's fallback font — or run `gaem fonts install` from a shell. Open a new terminal window afterwards.
- A capstone perk you've unlocked but never taken now holds a slot on every level-up card until you claim it. Fate's Hand rerolls the other two.
- Restore a World from a backup on the Worlds screen. A restore always creates a new World rather than overwriting one, so at the five-World cap you have to delete one first.
- The Patrons codex counts the finished runs you've bound to each patron. Existing Worlds start from zero.

### Changed
- The per-turn damage-over-time line now names its kinds — "Poisoned 3, Burning 1" rather than a bare total. Still one line.
- Never seen a forge before? You might now.
- The dungeon is a bit friendlier to capitalists.
- Rune fragments drop as piles on the ground; pick them up with `g`. The pouch stacks them by type — up to 7 types — and a fragment it can't take is left where it lies rather than destroyed.
- Fragment drops retuned.
- Permanent upgrades wear down. A decayable line you haven't bought in 20 completed runs loses a tier, never below tier 2; re-buying restores it at the current price. Capstone unlocks stay yours once bought.
- Starvation has been revamped.
- The meaning of Skulls is hopefully less confusing.
- Settings takes ↑↓ and Enter now. Every row's letter shortcut still works.
- The feedback box's message editor has a cursor — move around inside what you've typed instead of backspacing to it.
- Worlds generate differently from seeds in this version.
- A run saved by this version cannot be resumed by an older one. The older build backs the save up and starts fresh, and does not hand it back if you upgrade again.

### Fixed
- Class-veteran achievements unlock again. Wins you recorded before this fix count retroactively — the unlock lands at the end of your next completed run.
- Vampir lifesteal drains the physical hit, not just rune-proc damage — it had been counting only the procs. The Triune adds another 8 points on top.
- A trap you've already sprung stops reading as armed once it's off-screen — it went on showing red from memory. Spotted-but-unsprung traps now stay on the map after you walk away, too.
- The forge takes ↑↓ and Enter, like every other picker.
- The Worlds create-name hint fits inside its box.
- The merchant's sell list stays inside a short terminal.
- The deep upgrades heading no longer advertises a shard cost it doesn't charge.

## [v0.12.0] - 2026-07-26

### Added
- **Two new backgrounds.** The Forager works the ground as they walk it. The MAHA member eats what they kill, and pays for it elsewhere.
- **A new patron** keeps a shrine, with a boon for the hungry and a toll for the wealthy.
- **Oaths are now something you buy.** Contract slots and a once-per-run oath reroll join the permanent upgrades. **Contracts are no longer offered for free — in an existing save you will need to buy your first slot before any run offers an oath again.**
- **More at the top of the mastery ladder** — the last five rungs carry class kits, quirks, grave legacies, shrine perks, and a pinnacle for each class.
- **Merchants offer services** now, including a paid reroll of their stock.
- **A damage tally** in the message log — press Tab to see what has actually been hurting you.
- Enemy pathfinding updated.

### Changed
- **Some things that were free now take a turn** — equipping and unequipping, studying an item you already own, and doing business with merchants, forges, and shrines. Enemies act while you do. Refusing or cancelling still costs nothing.
- Worlds generate differently from seeds in this version.
- Long runs are now shared in truncated form instead of being dropped for being too large.

### Fixed
- **The character sheet scrolls.** Everything past 32 rows was being dropped silently, so Pacts — the only place to read what a pact does to your resistances — disappeared once you had four perks, which is an ordinary level-12 build.
- **Spike-trap damage now counts.** It was never added to your run's total, so "Damage Endured" undercounted every trap you stepped on, and a trap that left you at a sliver of health never counted toward Survivor.
- A turn spent behind an open bag or item card no longer hides what happened — the overlay closes if the turn drew blood.

## [v0.11.0] - 2026-07-24

### Added
- **Consumables now stack in a dedicated pouch** shown beside your bag, with cursor navigation across both. Merchants sell consumables straight into it and you can sell from its stacks.
- **Quick-quaff** — `q` drinks the best health potion you have with the least amount of overheal.
- **Class mastery expanded** — there's more to discover in the codex for veterans of each class.
- **Essence updates** — class masteries require essence to purchase once unlocked. **In an existing save, you will need to purchase anything already unlocked before it applies again.** There are new things to spend your essence on, and essence earnings and costs have been retuned.
- **Gold changes** — gold earnings have been retuned, prices have been retuned, and there are new things to discover that will help you empty your purse.
- **Potions** — health and mana potions have been revamped, with stronger late-game heals priced to match, and the merchant's 'randomly stronger potion' slot has been removed.
- Traps you place now stop a charging enemy in its tracks, and spring when an enemy teleports onto them.
- The character sheet shows live counts of your passive effects firing this run, and identifies heirloom passives.
- A proc-glyph overlay flags passive effects the turn they fire — toggle it in Settings.
- The run summary gains a Build Report: what your perks, runes, and passives actually did all run.
- The death screen and run summary now name what killed you.
- The run summary names your runes — sigils, names, or both, via a new setting.
- Settings has a language picker (also settable via `GAEM_LOCALE`).
- You can back up a World from the Worlds screen. Old backups aren't pruned automatically. The game keeps a damaged save rather than erasing it and can usually rebuild your progress from your logs, but a backup is still a good idea. Once created, the game won't touch the backup again.

### Changed
- Worlds generate differently from seeds in this version.

### Fixed
- The Frail mutation's −25% max HP penalty now actually applies — it was wiped at run start, so Frail runs were collecting bonus essence at no cost.
- The Vitality upgrade and class-mastery HP/MP bonuses now actually apply — they were also wiped at run start, since the beginning.
- Gold spending was counted twice at merchants, so the Big Spender achievement fired at about half its real target. It counts once now, and descent taxes and forge fees now count as spending too; already-inflated lifetime counters keep their value.
- The achievements codex keeps an achievement's goal text after you unlock it.
- The drop-confirmation box now fits narrow terminals.
- Weapon effects are tied to the item itself, so renamed items (locale overrides) keep their procs.
- Ring of Wisdom and Ring of Insight had their stat bonuses swapped — Wisdom now grants WIS and Insight grants INT. A ring already in your bag keeps its old bonus until it's replaced.
- `gaem recover-meta` now rebuilds far more of your progress from recent logs — essence and shard balances, mutations, triunes seen, nemesis kills, and floor counters — where it previously reset them to zero.
- The merchant buy screen no longer clips a bought-back heirloom's badge and price, and now fits narrow terminals.

## [v0.10.0] - 2026-07-19

### Added
- **gaem can update itself.** The update overlay (`[U]`) now downloads, verifies, and installs a new release in place — no more manual download-and-replace. If anything fails it rolls back and drops you to the manual steps, unchanged. You'll install this release the usual way; every update after it can be applied in-game.
- **Optional auto-update on launch.** A new Settings row (`[D]`) applies a pending update the next time you start the game. Off by default.
- **Update-check notifications can now be toggled from Settings** — previously this was only controllable through an environment variable.
- **Stable builds can opt into release-candidate offers**, if you want updates before they land as a stable release.

## [v0.9.0] - 2026-07-18

### Added
- **A seventh patron: the Dust-Crown.** Its pact trades a boon that grows with your Pantheon for an essence tithe that never lifts.
- Feedback that fails to send now queues and can be resent with `r` from the feedback wizard; a successful send also delivers anything still queued.
- The Contracts tab tracks your lifetime contract record.

### Changed
- Contracts pay out the moment you complete them, plus a small bonus for finishing all three in a run.
- Contract offers lean toward how you actually play, and scale with Ascension.
- Heirloom names show their most recent owner with a generation marker; the examine card keeps the full chain.
- Food prices now differ by item — an apple costs less than elven waybread. Sell prices unchanged.
- The feedback wizard redacts logs before sending and caps attachments at 4 MiB.
- The run summary fits short terminals by trimming optional sections.
- Worlds generate differently from seeds in this version.
- Text overrides: boss dialogue keys are now `boss.<slug>.dialogue` (was `boss.<floor>.dialogue`) — update override files.

### Fixed
- Traps you place now damage enemies.
- The poison immunity perk now actually prevents poison.
- Dropping an item no longer closes the drop picker unless the turn hurt you.
- Near-deaths from enemy abilities and explosions now count toward the Survivor achievement; older saves catch up as the counter ramps.
- Bag and drop pickers scroll on short terminals instead of overflowing, and the ending screen reflows to stay on screen.
- A merchant row you've selected but can't afford is now readable.
- Several screens no longer show broken characters or misaligned text with long or accented names (sell list, examine card, forge, codex, sidebar contracts, Worlds screen).

## [v0.8.0] - 2026-07-16

### Added
- Browse the merchant's shop: scroll through the stock and your sell list with the arrow keys, and examine any item before buying or selling.
- The run summary lists each contract with its reward and progress.
- Bag capacity is now 20 slots, up from 15.

### Changed
- **Combat rebalance.** Armor now reduces damage proportionally instead of subtracting a flat amount — defense always matters, but never shrinks a hit to nothing. Enemies chip harder, especially bosses and on the early floors; heavily-armored enemies take a little less from you.
- Enemies scale more smoothly as you descend (floors 2–3 ease off slightly). Elites hit noticeably harder, Tough elites gain real armor, and the floor 6, 9, and 12 bosses have more HP.
- Loot rebalanced: less gear, more potions and gold. Floors 1–3 spawn more items, and a fresh run starts with 4 rations instead of 3.

### Fixed
- Five upgrade capstones (Provisions, Keen Eye, Merchant's Favor, Second Wind, Midas Touch) could be bought but did nothing. All five now work.
- Rings now grant what they say: Rings of Wisdom, Insight, and Fortune did nothing at all, and a Constitution ring never gave its max HP. Every ring is now worth exactly its printed stat points — which also means Ring of Protection's defense drops from +3 to +1.
- Floors 14–15 no longer drop only the weakest gear.
- The death screen no longer swaps its parting words on every keypress, and they match the message log.
- Potion text shows the amounts you'll actually get with your bonuses applied.
- A line of forge flavor text that could never fit on screen now appears.
- `gaem recover-meta` no longer over-counts floors by one per won run, and it now reconstructs your lifetime relic shards. (Older reports advised zeroing the floors counter — that advice was wrong and is gone.)

## [v0.7.0] - 2026-07-15

### Added
- Opt-in run-log sharing: after a run, you can choose to start sharing a redacted slice of your run logs to help with balance and bug fixes. Off by default, switchable anytime from the new Settings screen (`[S]`).

### Fixed
- Fixed long item, skill, and gear names occasionally showing a broken character when truncated, and a merchant price-delta tag that could drift out of place.
- Named heirloom weapons (like the Watch Warden's Blade) now keep their name and bonus when inherited, instead of being renamed with a random bonus.
- Fixed the Deep Explorer and Treasure Hunter achievements never tracking progress.
- Fixed five achievements (Deathless, Starving Artist, Pacifist, Combo King, Untouched) that were impossible to earn due to untracked counters.

## [v0.6.1] - 2026-07-15

### Fixed
- Fixed the game not starting on Windows. v0.6.0 crashed on launch before reaching the menu; Linux and macOS were unaffected. No save data or progress is touched.
- **The feedback consent screen now describes the attached log correctly.** It said it would attach "this run's log"; the log actually covers your whole session — every run since you launched the game. This is a wording fix, not a behavior change: the log always covered the full session, and nothing about what is collected or sent has changed. The screen was wrong; the data is the same.
- Fixed frame titles and the death-summary line rendering wrong with accented or non-Latin characters.

### Changed
- Inventory rows now abbreviate heirloom passives and shorten long item names so the passive and free rune slots stay visible. The full name, passive and description are on the examine card.

## [v0.6.0] - 2026-07-14

### Added
- **Inventory rework.** Move a cursor through your bag with the arrow keys, examine any item in place, and unequip straight to the bag. Letter keys still work as before.
- **Mouse-wheel scrolling** in the message log.
- **Text overrides.** Every player-facing string now loads from data files you can edit; `gaem dump-strings` writes out a copy to start from. English only for now.

### Changed
- The bag sorts itself — category first, then value.
- `i` closes the inventory as well as opening it. Bag slots skip the letter `i`.
- Flavor text reworked across the game: floor arrivals, boss lines, patron and rune lore, and character backstories now vary between runs.
- Floors shift in color as you descend through a region.
- A one-time update to enemy knowledge on first launch — your bestiary progress carries over.

### Fixed
- Summoned minions no longer pay out XP, essence, gold, loot, or contract progress. Bestiary knowledge still counts.
- Gold picked up off the floor now counts toward your run totals.
- Fixed swap-on-pickup offering an equipped item the new item couldn't replace, which could wedge an unreachable slot in the bag.
- `gaem recover-meta` rebuilds Regard kills, all-seals runs, and true-ending victories.

## [v0.5.1] - 2026-07-12

### Fixed
- Fixed a new run starting with the previous character's state after a death in the same session.
- Fixed the feedback box's character counter hiding the end of the message.
- Fixed lowercase 'c' not working on meta menu tabs when a saved run exists.

## [v0.5.0] - 2026-07-12

### Added
- **Sealed Vaults.** Certain floors now hide a sealed gate leading to an optional side chamber — a self-contained, high-risk detour with its own dangers and rewards. Clearing enough of them opens a new path to a true ending.

### Fixed
- **Several mutations now work as intended.** A handful of selectable mutations had no effect when chosen; they're now fully wired up.

## [v0.4.0] - 2026-07-05

### Added

- **Revamped dungeon floor generation.** The descent now passes through multiple distinct regions, each with its own architecture and colors, plus new terrain hazards.

### Changed

- **Rebalanced enemy and loot density across the deeper floors.**

## [v0.3.4] - 2026-06-15

### Added

- **In-game update check.** The meta menu now shows a one-line banner when a newer release is available, with a text-only **[U]** overlay that links the release page and walks you through manual install. The check is a once-a-day, read-only call to the GitHub releases API — it sends no player data, is skipped for development builds, and can be turned off entirely with `GAEM_NO_UPDATE_CHECK=1`.

## [v0.3.3] - 2026-06-14

### Fixed

- **Feedback submissions now include your run log.** When you choose to attach your run log to in-game feedback, the log file is now actually delivered with the message — previously it was silently dropped, so reports arrived without it.
- **The feedback consent screen no longer promises a run log it can't send.** On the meta menu (or very early in a run, before anything is logged) the screen used to say your run log would be attached even when there was nothing to send; it now offers the attachment only once the log has content.

## [v0.3.2] - 2026-06-14

### Fixed

- **Merchants are no longer easy to mistake for a pile of gold.** The merchant now appears as a distinct shopkeeper's house (⌂) in aqua, instead of the gold-colored `$` it shared with gold piles.
- **Merchants you've already found stay marked on the map.** A merchant now remains visible (dimmed) on your explored map after you walk away, like walls and stairways, instead of vanishing the moment it leaves your view.

## [v0.3.1] - 2026-06-14

### Fixed

- **In-game feedback wizard now works in a live terminal.** The feedback wizard (press **Z**) rendered but never repainted as you moved through its steps, so it looked frozen and couldn't be used — the v0.3.0 feedback feature now updates the screen each step and responds to input as expected.

## [v0.3.0] - 2026-06-14

### Added

- **In-game player feedback.** Press **Z** anywhere — meta menu or mid-run — to open a 4-step feedback wizard and send a note straight to the developers. Submission is asynchronous (it never blocks the game) with an **offline fallback**: if it can't reach the network your feedback is saved locally so nothing is lost.
- **In-game changelog.** A new meta-menu tab shows the release notes for the build you're running.
- **Build version on the meta menu.** The running build's version is now shown on the meta menu.
- **Sell any inventory item, with buyback.** The merchant sell flow now opens a picker so you choose exactly which inventory item to sell; sold items re-enter the merchant's stock at full buy price, so you can buy them back. No arbitrage — you pay the normal buy/sell spread.
- **Merchant purchase confirmation.** Buying from a merchant now asks for a Y/N confirmation before the gold leaves your purse.
- **Equipped-vs-stock stat deltas at the merchant.** The merchant overlay shows how each stocked item compares, stat by stat, to what you currently have equipped.
- **Numpad diagonal movement.** Diagonal steps now work from the numeric keypad (7/9/1/3, NumLock off).

### Changed

- **Gentler early-game hunger.** Fresh profiles start with more rations, ground food drops more often, merchants stock food, and hunger drains more slowly — early-game food pressure is loosened while the hunger-vs-exploration tension remains.

### Fixed

- **Quitting to the menu mid-run no longer exits the game.** Choosing quit-to-menu during a run now returns to the meta menu instead of closing gaem entirely.

## [v0.2.0] - 2026-06-12

### Added

- **Pantheon & victory monuments.** Victorious characters are enshrined in the Pantheon; a new **Pantheon tab** lists ENSHRINED victors and THE FALLEN. **Victory monuments (▲)** spawn on descending floors of later runs once you have a victor — collecting one grants a gold tithe, a category gear roll, and rune fragments.
- **A richer legacy for the fallen.** Dead characters now carry **names**, leave **visitable graves** with map markers, can return as **class-themed nemesis variants** that grant a **boon on kill**, and pass on **inherited weapons plus 6 legacy passives**.
- **Multiple save worlds.** The meta menu gains **[W] Worlds** — up to 5 independent progression worlds with create / switch / delete; **[P]** jumps to the Pantheon tab.
- **Per-turn autosave.** Runs autosave every turn, so a crash or power loss resumes at the exact turn via **[C] Continue** (death and victory still delete the save by design).

### Changed

- **Save data layout.** Saves now live under `profiles/default/`; the first launch after upgrading silently migrates your save data there (lossless, mid-run-safe). One-time migration.
- **Corrupt-save handling.** A corrupt or newer-version `meta.json` is now backed up to a timestamped `.bak` with an on-screen notice instead of being silently reset; older saves load unchanged.
- **Atomic save writes.** Save files are written atomically; a present-but-unreadable `meta.json` is backed up before the next save replaces it.

### Fixed

- **survivor / ascendant max-HP rewards now apply.** Both were silently wiped by stat recalculation (survivor's +5 was dead since launch; ascendant's +10% died on first recompute). Players with either achievement unlocked will see their max HP genuinely rise for the first time — expected correction, not a buff.
- **Grave markers no longer crowd onto adjacent tiles.**
- **No more stale legacy grave reference** when the last fallen entry is evicted from an all-victor archive.

## [v0.1.6] - 2026-06-12

### Fixed

- Floor-count achievements (Cartographer and the other floors-descended milestones) now progress. The "floors descended" counter was stuck at zero, so these achievements never advanced. Characters with pre-fix history will see them unlock late as the counter catches up — expected retroactive catch-up, not a bug.
- `gaem agent-playtest` no longer corrupts your saved data. The autonomous-run harness was writing to your real save file and deleting your in-progress run; harness runs are now fully sandboxed and never touch live saves.

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
