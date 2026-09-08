# TheatreMix `.tmix` — Agent Reference

Purpose: everything needed to implement a **correct read-only parser** for TheatreMix show files,
with no other source. Derived from TheatreMix **3.5.0**. Rationale and evidence live in the
human-oriented `TMIX_FORMAT_SPEC.md`; this file states facts only.

Tags: `[C]` confirmed · `[I]` inferred (implement, but tolerate deviation) · `[?]` unknown
(**preserve the raw value; do not interpret**). Untagged statements are `[C]`.

---

## 0. Ten facts to hold in context

1. A `.tmix` file is a **plain SQLite 3 database**. No encryption, no compression,
   `application_id = user_version = 0`. Writer: Qt `QSQLITE`.
2. Identify by presence of tables **`config`, `cues`, `profiles`**.
3. All structured values are TEXT micro-formats: `IntList` `1,2,3` · `IntMap` `k=v,k=v`
   (**unordered**) · `PlusList` `3+4` · `SemiList` `a;b`. No quoting, no escaping, no spaces.
4. **`NULL` ≡ `""`** everywhere. Tables, columns and config params may be **absent** (lazy
   migration) ⇒ treat as empty / default. **Address columns by name, never by ordinal.**
5. Cue identity = `(number, point)`. Display `N` when `point=0`, else `N.P` with **no zero
   padding**; `point` is an integer suffix, so `0.1 ≠ 0.10`. Sort numerically on the tuple.
   Cue `(0,0)` is implicit and **never stored**.
6. Channel ids: positive = console input channel; **negative `-n` = Aux In n**.
7. Channel names live in `profiles` rows with `default=1` — exactly one per entry of
   `config.channels`. There is no channel table.
8. Per-cue maps (`channelPositions`, `channelProfiles`, `channelFX`, `channelLevels`) contain
   **only non-default entries**, and **only for channels assigned to a DCA in that cue**.
9. `cues.fxMutes` lists FX buses to **unmute** (misnomer). `channelLevels` unit is **0.1 dB**.
10. There is no serialisation library to interoperate with: values are built with Qt
    `QString::arg("%1=%2,")` + `chop(1)` and read with `QString::split`.

---

## 1. Open and validate

```
open(path, read_only)                       # "?mode=ro" or "immutable=1"
tables  = {name FROM sqlite_master WHERE type='table'}
REQUIRE {'config','cues','profiles'} ⊆ tables      # else: not a .tmix
cols[t] = [name FROM PRAGMA table_info(t)] for t in tables
cfg     = {param: value FROM config} merged over DEFAULTS (§3.1)
```

Column access rule: `row[col] if col in cols[table] else None`; then normalise `None` and `""`
to the empty value of the column's format.

Versioning: `cfg.minVersion` (str, e.g. `3.0`) = minimum app version that may open the file —
read, do not enforce. `cfg.profileSchemaVersion` (`2` in all known files; `1` legacy) versions
`profiles.data`.

---

## 2. Primitives

```ebnf
Int       = ["-"] DIGIT+ ;
Bool      = "0" | "1" ;
IntList   = [ Int { "," Int } ] ;                 (* set semantics unless stated *)
IntMap    = [ Int "=" Value { "," Int "=" Value } ] ;   (* keys unique; order arbitrary *)
PlusList  = Int { "+" Int } ;                     (* value type inside channelFX *)
SemiList  = WORD { ";" WORD } ;                   (* inside profiles.data only *)
Empty     = NULL | "" ;                           (* == empty list / empty map / unset *)
```

Reference implementation (authoritative):

```python
def int_list(s: str | None) -> list[int]:
    return [int(t) for t in s.split(',') if t] if s else []

def int_map(s: str | None, value=int) -> dict[int, object]:
    out = {}
    for tok in (s or '').split(','):
        if tok:
            k, v = tok.split('=', 1)
            out[int(k)] = value(v)
    return out

def fx_spec(v: str) -> int | list[int]:           # channelFX value
    return -1 if v == '-1' else [int(t) for t in v.split('+')]

def text(s: str | None) -> str:
    return s or ''
```

---

## 3. Tables

### 3.1 `config` — `(param TEXT PK, value TEXT)`

Read into a dict; apply defaults for missing params. Types are the *interpretation* of the TEXT.
`console-specific` = value chosen from `targetConsole` at creation; no universal default.

| param | type | default | meaning |
|---|---|---|---|
| `designer` | str | `""` | Designer name |
| `venue` | str | `""` | Venue name |
| `minVersion` | str | app-specific | Min app version to open (§1) |
| `profileSchemaVersion` | int | `2` | Grammar version of `profiles.data` |
| `targetConsole` | str | set at creation | Console family for UI. Known codes: `X32 X32C X32P X32RACK X32CORE M32 M32R M32C WING WINGC SQ-5 SQ-6 SQ-7 TF1 TF3 TF5 QL1 QL5 CL1 CL3 CL5 GLD-80 GLD-112 Avantis Avantis-Solo DM32 DM0 DM48 DM64 CDM32 CDM48 CDM64 DM7 DM7C Qu-5 Qu-6 Qu-7 CQ…` |
| `consoleModel` | str | `""` | Last connected model code (may differ from `targetConsole`) |
| `consoleVersion` | str | `""` | Last connected firmware version |
| `consoleIP` | str | `""` | Last connected IPv4 |
| `consoleMAC` | str | `""` | Last connected MAC |
| `autoConnect` | Bool | `0` | Auto-connect on open |
| `channels` | IntList (ordered) | `""` | Controlled channels in console order; ≤ 48; negatives = Aux In |
| `dcas` | IntList (ordered) | `""` | Controlled console DCA numbers; `len` ≤ 12 ⇒ which `dcaNN*` columns matter |
| `backupChannels` | `[?]` | `""` | Fixed backup per primary channel; shape unobserved (probably IntMap primary→backup). Keep raw. |
| `spareBackup` | Int | `0` | Spare backup channel; `0` = none |
| `fxAssigns` | IntList | `""` | FX buses (1–4) assignable to channels |
| `fxMutes` | IntList | `""` | FX buses whose mutes are cue-programmed |
| `defaultFX` | Int | `-1` | FX bus for channels lacking an explicit `channelFX` entry; `-1` = none |
| `fxBusMap` | IntMap fx→busId | `1=13,2=14,3=15,4=16` (X32/M32); console-specific | TheatreMix FX n → console bus id (§4). "Inhibited" encoding `[?]` |
| `channelLevels` | Bool | `0` | Level-offset feature on |
| `cueZeroResetLevels` | Bool | `0` | Cue 0 forces channel faders to 0 dB |
| `cueZeroActorLabels` | Bool | `0` | Actor names on scribbles in `0.x` cues |
| `snippetRecall` | Bool | `0` | Snippet recall feature on |
| `sceneRecall` | Bool | `0` | Scene recall feature on |
| `cueZeroSnippets` | IntList | `""` | Snippets fired by implicit cue 0 |
| `cueZeroScenes` | IntList | `""` | Scenes fired by cue 0 |
| `cueZeroScenePoints` | IntList | `""` | Parallel to `cueZeroScenes` (§3.2 `scenePoints`) |
| `qLabCues` | Int | `0` | Playback recall: `0` off, `1` QLab, `2` Show Cue System `[I]`, `3` Cue Player `[I]` |
| `qLabPasscode` | str | `""` | QLab OSC passcode |
| `qLabSuppressBack` | Bool | `0` | Don't re-fire playback cue on Back |
| `dawRemote` | Bool | `0` | REAPER link on |
| `dawIP` | str | `""` | REAPER IP |
| `dawPreRoll` | Int | `0` | REAPER pre-roll seconds |
| `gangLR` | Bool | `0` | LR-fader ganging (X32/M32) |
| `gangLRChannels` | token list | `""` | Tokens match `^((\d{1,2})\|(-\d)\|(Fx \d(L\|R))\|(Bus \d{1,2}))$` (input, aux-in, FX return, mix bus); comma-separated `[I]`; unobserved |
| `gangLRName` | str | `""` | LR scribble label |
| `gangLRColour` | Int or `""` | `""` | Console colour index for LR scribble |
| `labelLR` | Int | console-specific | Show cue info on a scribble strip: `0` off, `1` on |
| `labelTargetBus` | busId | console-specific; `1000` = LR | Strip that shows cue info (LR or a non-controlled DCA) |
| `consoleMuteDCAUnassign` | Bool | `1` | Channel mute button unassigns from DCA while editing |
| `suppressDCAMuteBackupSwitch` | Bool | `0` | Suppress DCA-mute backup switching |
| `selectOnSpill` | Bool | `1` | DCA-spill option; semantics `[?]` |
| `enableChannelMonitoring` | Bool | `1` | Silence/clip monitoring |
| `activeChannelHighlight` | Bool | `0` | Yellow scribbles on active channels |
| `dimDCAFaders` | Bool | `0` | Dim inactive DCA scribbles |
| `dimDCAFadersSuppressColours` | Bool | `0` | Active DCAs shown white when dimming |
| `qlclDyn1` | Bool | `0` | Yamaha QL/CL/DM7: use dynamics slot 1 |
| `buttonMap` | IntMap action→button | console-specific (e.g. `0=12,1=11`) | Console assign button per action. Action `0`=Go, `1`=Back; indices ≥2 `[?]` (candidate order: QLab Go, Stop, Panic, Pause, Resume, SCS Go, Stop, Fade, PauseRes, CP Go, Stop, Fade, Pause, Resume, Mark, RecOffset, CloneOffset, REAPER Play/Pause, REAPER Stop) |
| `muteButtonMap` | IntMap action→muteGroup | `""` | Same action indices → mute-group button 1–8 `[I]` |
| `muteButtonAssignKeys` | IntMap muteGroup→softKey | `""` | A&H soft-key number per mute group `[I]` |

### 3.2 `cues`

PK/unique: `(number, point)`. One row per stored cue.

| column | format | empty ⇒ | semantics |
|---|---|---|---|
| `number` | Int 0–9999 | — (NOT NULL; DDL default 999 is a placeholder) | Cue number |
| `point` | Int 0–99 | — (NOT NULL, default 0) | Integer suffix; `0` = whole cue |
| `name` | str | `""` | Cue text. Leading `>` (repeatable) = indent |
| `dcaNNChannels` NN=01..12 | IntList (set) | unassigned | Channels on DCA NN. Every controlled channel absent from all DCAs is **muted** by the app. Columns NN > `len(cfg.dcas)` are unused |
| `dcaNNLabel` | str | derived by app | Scribble label. Non-empty with empty channels = placeholder DCA. Ensemble name is stored here when an ensemble was typed |
| `channelPositions` | IntMap ch→positionId | all default (0) | Non-zero positions only |
| `channelProfiles` | IntMap ch→profileId | all default | Non-default profiles only; `profiles.channel == ch` |
| `channelFX` | IntMap ch→FxSpec | all `cfg.defaultFX` | FxSpec: `-1` none · `n` · `n+m…` |
| `fxMutes` | IntList | none unmuted | FX buses **unmuted** in this cue; others in `cfg.fxMutes` muted |
| `snippets` | IntList | none | Console snippet indices to recall `[I: format]` |
| `scenes` | IntList | none | Console scene numbers to recall |
| `scenePoints` | IntList | none | Parallel to `scenes`, same length; decimal part (Yamaha `x.yy`), `0` otherwise `[I: multi-element]` |
| `qLabCue` | str | none | Playback cue number (exact string; case/whitespace significant) |
| `channelLevels` | IntMap ch→Int | none | Fader offset in **0.1 dB**; range −150…+50; `0` never stored |
| `colour` | Int/NULL | none | `0`/NULL none; `1` red, `2` yellow, `3` green, `4` blue, `5` purple `[I: order; 0=none is C]` |
| `skip` | Int/NULL | `0` | `1` = skipped on Go |

Cue numbering rules:
- display = `str(number)` if `point == 0` else `f"{number}.{point}"`
- order   = `(number, point)` ascending, numeric ⇒ `0.9 < 0.10`
- `(0,0)` never stored; it is the fixed line-checks cue (all DCAs clear, defaults everywhere,
  fires `cfg.cueZeroSnippets/Scenes/ScenePoints`). `0.x` rows are ordinary stored cues.

### 3.3 `positions`

| column | format | semantics |
|---|---|---|
| `id` | Int PK | `0` = built-in default (`Centre Stage`/`CS`), always present |
| `name` | str | Display name |
| `shortName` | str | Cue-list code; uppercase alphanumeric |
| `delay` | Int (NUMERIC) | ms |
| `pan` | Int (NUMERIC) | negative left · `0` centre · positive right (console pan %) |
| `buses` | IntList of busId / absent | Bus sends unmuted at this position; all other positions' buses muted |

### 3.4 `profiles`

| column | format | semantics |
|---|---|---|
| `id` | Int PK | |
| `channel` | Int | Channel (may be negative) |
| `name` | str | `default=1` ⇒ **the channel's name**; else the profile name |
| `label` | str / absent | Optional short scribble label |
| `default` | Int | `1` = base profile; exactly one per controlled channel |
| `data` | ProfileData / empty | See grammar |

```ebnf
ProfileData = [ Pair { "," Pair } ] ;
Pair        = Path "=" Scalar
            | "storeParams" "=" SemiList        (* ⊆ gain;hpf;eq;dyn, in that order *)
            | "actorParams" "=" SemiList ;
Path        = "/" SEG { "/" SEG } ;             (* OSC-style, channel-relative *)
Scalar      = NUMBER | TOKEN ;                  (* no ',' inside *)
```

Example: `/gain/headamp=5,/gain/trim=8.9,/hpf/f=100,/hpf/on=1,/eq/1/f=190,/eq/1/g=-9.4,/eq/1/q=0.77,/eq/1/type=PEQ,…,/eq/on=1,storeParams=gain;hpf;eq`

Known paths (do not assume closed set): `/gain/headamp` `/gain/trim` `/hpf/f` `/hpf/on` `/eq/on`
`/eq/{1..6}/{f,g,q,type}` `/dyn/{on,thr,ratio,knee,mgain,att,attack,hld,hold,rel,release,pos,keysrc,mix,auto,mode,det,env,gain,mdl,filter/on,filter/type,filter/f}`
`/preamp/{hp,hpf,hpon,trim}` `/delay` `/backup/{gain,hpf,eq,dyn}/…` `[I]`.
EQ type tokens: `PEQ LShv HShv LCut HCut` (+ console-specific e.g. `VEQ BU BS`).
Parse as `dict[str, str]`; convert on demand. `storeParams`/`actorParams` = which processing
types are stored; a profile may carry only those keys.

### 3.5 `ensembles`

| column | format | semantics |
|---|---|---|
| `id` | Int PK | `1` = implicit "All" (**never stored**); `2` Male, `3` Female pre-created (may be empty) |
| `name` | str | |
| `channels` | IntList | Members |
| `channelProfiles` | IntMap ch→profileId / absent | Profile applied to member on assignment |

Cues store expanded channels; ensembles are not referenced from cues.
DDL quirk: `channels` has declared type ``TEXT`channelProfiles` TEXT`` (missing comma). Harmless
when addressing by name; may confuse tools that parse declared types.

### 3.6 `actors` / `actorProfiles` / `actorGroups`

| table | columns | semantics |
|---|---|---|
| `actors` | `id` PK, `channel` Int, `name` str, `order` Int, `active` Bool | Performer on a channel; `active=1` = current |
| `actorProfiles` | `(actor, profile)` unique, `data` | Per-actor-per-profile ProfileData `[I: grammar]` |
| `actorGroups` | `id` PK, `name`, `data` IntMap ch→actorId `[I]` | A cast: active actor per channel |

Dangling actor ids in `actorGroups` occur (actors deleted) — not an error.

### 3.7 Caches (optional tables, informational)

| table | key | value |
|---|---|---|
| `snippetCache` | `snippet` Int (console index) | `name` |
| `fxCache` | `fx` Int (TheatreMix FX 1–4) | console FX slot `name` |
| `sceneCache` | `scene` Int, `point` Int/absent | `name` |

`sqlite_sequence`: ignore (ids never reused; gaps normal).

---

## 4. Identifier domains and enumerations

| Domain | Rule |
|---|---|
| Channel | `> 0` input channel (≤128 on dLive); `< 0` ⇒ Aux In `-n` (X32/M32 1–6, 7/8 = USB L/R); `0` unused |
| DCA | 1–12 via `dcaNN` columns; console numbering |
| Position id | `positions.id`; `0` default |
| Profile id | `profiles.id` |
| Ensemble id | `1` implicit All; `2`,`3` defaults |
| FX bus | single digit 1–4 (TheatreMix-side); console bus via `cfg.fxBusMap` |
| Bus id | X32/M32: mix bus 1–16. Others: ≥1000 codes (`1000` LR; `1101–1104` dLive FX sends; `1302` a position bus; `1408` a WING DCA). Encoding `[?]` — keep raw |
| Scene number | X32 0-based; TF `0–99`⇒A00–A99, `100–199`⇒B00–B99; QL/CL `x.yy` ⇒ (`scene`,`point`) |
| Colour | `0`/NULL none, `1` red, `2` yellow, `3` green, `4` blue, `5` purple `[I]` |
| Action index | `0` Go, `1` Back; others `[?]` |
| Playback (`qLabCues`) | `0` off, `1` QLab, `2` SCS `[I]`, `3` Cue Player `[I]` |

---

## 5. Derived values (implement as methods)

```
default_profile_id[ch]      = id of profiles row with channel==ch and default==1
channel_name[ch]            = profiles[default_profile_id[ch]].name
cue.assigned                = ⋃ cue.dca[n].channels
cue.muted                   = set(cfg.channels) − cue.assigned
cue.position(ch)            = cue.channelPositions.get(ch, 0)
cue.profile(ch)             = cue.channelProfiles.get(ch, default_profile_id[ch])
cue.fx(ch)                  = cue.channelFX.get(ch, cfg.defaultFX)      # -1 ⇒ none
cue.level_db(ch)            = cue.channelLevels.get(ch, 0) / 10.0
cue.fx_muted                = set(cfg.fxMutes) − set(cue.fxMutes)      # if feature enabled
cue.display                 = f"{number}.{point}" if point else str(number)
console_bus(fx)             = cfg.fxBusMap[fx]
```

Feature-enabled gating (data persists when a feature is toggled off):
`channelLevels`→`cfg.channelLevels`, `snippets`→`cfg.snippetRecall`, `scenes`→`cfg.sceneRecall`,
`qLabCue`→`cfg.qLabCues≠0`, `fxMutes`→`cfg.fxMutes≠[]`. Parse always; expose the flag.

---

## 6. Invariants (assert in tests; all held on ~800 real cues)

```
I1  ∀ cue, ∀ n: cue.dca[n].channels ⊆ cfg.channels
I2  ∀ cue: DCA channel sets are pairwise disjoint
I3  ∀ cue: keys(channelPositions ∪ channelProfiles ∪ channelFX ∪ channelLevels) ⊆ cue.assigned
I4  ∀ cue: 0 ∉ values(channelPositions); values ⊆ positions.id
I5  ∀ cue, (ch→pid) ∈ channelProfiles: profiles[pid].default == 0 ∧ profiles[pid].channel == ch
I6  ∀ cue, v ∈ values(channelFX): v == -1 ∨ all(x ∈ cfg.fxAssigns for x in v)
I7  ∀ cue, v ∈ values(channelLevels): -150 ≤ v ≤ 50 ∧ v ≠ 0
I8  ∀ cue: cue.fxMutes ⊆ cfg.fxMutes
I9  |{p : p.default==1 ∧ p.channel==ch}| == 1  ∀ ch ∈ cfg.channels
I10 ¬∃ cue with (number, point) == (0, 0)
I11 (number, point) unique (DB index)
```

---

## 7. Gotchas (ordered by likelihood of causing a bug)

1. Reading columns by index or assuming a column exists.
2. Treating `IntMap` order as meaningful, or `IntList` DCA channels as sorted.
3. Interpreting `point` as a decimal (`"1.5"` ≠ `"1.50"`; `1.9 < 1.10`).
4. Treating `NULL` and `""` differently.
5. Reading `fxMutes` as the *muted* set — it is the *unmuted* set.
6. Forgetting the implicit cue 0 and implicit ensemble 1.
7. Applying `channelLevels` as dB instead of dB/10.
8. Expecting a channel name table — use `profiles.default=1`.
9. Assuming `channelFX` entries exist only when they differ from `defaultFX` — `-1` is stored
   explicitly even when `defaultFX=-1`.
10. Parsing `profiles.data` with a fixed key set — keys vary by console.
11. Choking on the `ensembles` DDL typo when introspecting declared types.
12. Splitting `channelFX` values on `,` before `=` — split pairs on `,`, then key/value on `=`,
    then value on `+`.

---

## 8. Unknowns and required handling

| Item | Handling |
|---|---|
| Bus-id encoding (non-X32) | Keep raw int; expose `is_x32_family` to interpret 1–16 as mix buses |
| `buttonMap`/`muteButtonMap` action ≥ 2 | Keep raw map; name only 0/1 |
| `backupChannels` | Keep raw string |
| `fxBusMap` "Inhibited" | Keep raw; do not assume all FX in `fxAssigns` have a mapping |
| `scenePoints` multi-element | Zip with `scenes`; if lengths differ, pad points with `0` |
| `colour` 3, `skip=1`, `profileSchemaVersion=1` | Unobserved; accept |
| `actorProfiles.data` | Parse with ProfileData grammar; tolerate failure |
| `selectOnSpill` | Bool passthrough |

---

## 9. Migration matrix (may be absent; treat as NULL / empty / default)

| Object | Optional members |
|---|---|
| `cues` | `channelFX snippets qLabCue channelLevels scenes colour scenePoints skip dca09Channels…dca12Label` |
| `profiles` | `label` |
| `positions` | `buses` |
| `ensembles` | `channelProfiles` |
| `sceneCache` | `point` |
| tables | `snippetCache fxCache sceneCache actors actorProfiles actorGroups` |
| `config` | any param (§3.1 defaults) |

Known variants: **A** (oldest) lacks `cues.scenePoints/skip`, `profiles.label`,
`sceneCache.point`, config `cueZeroActorLabels cueZeroScenePoints daw* dimDCAFadersSuppressColours
selectOnSpill`; **B** lacks `cues.skip`, config `cueZeroActorLabels daw* selectOnSpill`;
**C** = current.

---

## 10. Reference data model (Python, minimal)

```python
from dataclasses import dataclass, field

@dataclass
class DcaAssign:
    channels: set[int]
    label: str                     # "" ⇒ derived by app

@dataclass
class Cue:
    number: int
    point: int
    name: str
    dcas: dict[int, DcaAssign]     # 1-based, n ≤ len(cfg.dcas)
    positions: dict[int, int]      # ch → position id (non-default only)
    profiles: dict[int, int]       # ch → profile id (non-default only)
    fx: dict[int, int | list[int]] # ch → -1 | [bus, ...]
    levels: dict[int, int]         # ch → tenths of dB
    fx_unmuted: list[int]
    snippets: list[int]
    scenes: list[int]
    scene_points: list[int]
    playback_cue: str
    colour: int                    # 0 = none
    skip: bool
    @property
    def display(self): return f"{self.number}.{self.point}" if self.point else str(self.number)

@dataclass
class Position:  id: int; name: str; short_name: str; delay_ms: int; pan: int; buses: list[int]
@dataclass
class Profile:   id: int; channel: int; name: str; label: str; is_default: bool; data: dict[str, str]
@dataclass
class Ensemble:  id: int; name: str; channels: list[int]; channel_profiles: dict[int, int]
@dataclass
class Actor:     id: int; channel: int; name: str; order: int; active: bool

@dataclass
class Show:
    config: dict[str, str]         # raw, defaults applied
    channels: list[int]
    dcas: list[int]
    positions: dict[int, Position]
    profiles: dict[int, Profile]
    ensembles: dict[int, Ensemble]
    actors: dict[int, Actor] = field(default_factory=dict)
    actor_profiles: dict[tuple[int, int], dict[str, str]] = field(default_factory=dict)
    actor_groups: dict[int, tuple[str, dict[int, int]]] = field(default_factory=dict)
    snippet_names: dict[int, str] = field(default_factory=dict)
    fx_names: dict[int, str] = field(default_factory=dict)
    scene_names: dict[tuple[int, int], str] = field(default_factory=dict)
    cues: list[Cue] = field(default_factory=list)   # sorted by (number, point)
```

Cue row → `Cue` (all columns via the §1 access rule):

```python
def decode_cue(row, ndcas):
    dcas = {}
    for n in range(1, ndcas + 1):
        chs = int_list(row.get(f'dca{n:02d}Channels'))
        lbl = text(row.get(f'dca{n:02d}Label'))
        if chs or lbl:
            dcas[n] = DcaAssign(set(chs), lbl)
    return Cue(
        number=row['number'], point=row['point'], name=text(row.get('name')),
        dcas=dcas,
        positions=int_map(row.get('channelPositions')),
        profiles=int_map(row.get('channelProfiles')),
        fx=int_map(row.get('channelFX'), fx_spec),
        levels=int_map(row.get('channelLevels')),
        fx_unmuted=int_list(row.get('fxMutes')),
        snippets=int_list(row.get('snippets')),
        scenes=int_list(row.get('scenes')),
        scene_points=int_list(row.get('scenePoints')),
        playback_cue=text(row.get('qLabCue')),
        colour=row.get('colour') or 0,
        skip=bool(row.get('skip') or 0),
    )
```

---

## 11. Fixture coverage (what real data exists to test against)

Observed in real files: all three schema variants; X32/X32C, M32/M32R, WING/WINGC (target),
dLive CDM48/DM48, QL5, and a never-connected file; `dcas` 1–6 … 1–12 incl. populated
`dca09…12`; placeholder DCAs; negative channels; dLive channels to 88; `0.1` + `0.10`;
`channelPositions`, `positions.buses`, `channelProfiles` (cue + ensemble), `channelFX` with `-1`
and `n+m`, `fxMutes`, `channelLevels` at −150 and +50, `colour` ∈ {0,1,2,4,5,NULL}, `qLabCue`,
`scenes`+`scenePoints` (single), all three caches, `actorGroups`, one populated `profiles.data`
(dLive), non-default profiles, `spareBackup`, `gangLRName/Colour`, 4-action `buttonMap`,
`muteButtonMap`, `muteButtonAssignKeys`, `qLabCues=2`, `minVersion` 3.0/3.1.

Never observed (binary/help only): `cues.snippets`, `cueZero*` lists, `backupChannels`,
`gangLRChannels`, `profiles.label`, populated `actors`/`actorProfiles`, multi-element
`scenes`, `skip=1`, `colour=3`, `profileSchemaVersion=1`, inhibited `fxBusMap`, TF/SQ/Avantis/
DM7/GLD/Qu targets.
