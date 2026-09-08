# TheatreMix `.tmix` Show File — Format Specification

Reverse-engineered from **TheatreMix 3.5.0** (macOS, © Mixing Technology Pty Ltd) and a corpus of
real-world show files covering Behringer X32/M32, WING, Allen & Heath dLive and Yamaha QL consoles.
Sources of evidence:

- the sample show files themselves (schema, data, cross-file invariants);
- SQL statements, format strings, validation regexes and UI strings embedded in the
  application binary;
- the application's bundled help (feature guide and quick-start guide).

Confidence markers used throughout:

| Marker | Meaning |
|---|---|
| **[C]** | Confirmed by data and/or the application binary. |
| **[I]** | Inferred; the basis is stated. Safe to implement, but verify against new files. |
| **[?]** | Unresolved; treat the value as opaque. |

---

## 1. Container

| Property | Value |
|---|---|
| Container | Plain **SQLite 3** database file (magic `SQLite format 3\0`). **[C]** |
| Encoding | UTF-8. **[C]** |
| `PRAGMA application_id` / `user_version` | Both `0` — not used for identification. **[C]** |
| Writer | Qt `QSQLITE` driver (QtSql). Files seen were last written by SQLite 3.36.0–3.41.2. **[C]** |
| Compression / encryption | None. **[C]** |
| File extension | `.tmix`. The app also opens legacy `.x32tc` files (see Appendix C). **[C]** |
| Autosave | The app writes `<show>.recovery.tmix` recovery copies in the same format. **[C]** |

**Identification.** A file is a TheatreMix show if it is a valid SQLite database containing
tables `config`, `cues` and `profiles` (the binary checks these three names). **[C]**

**Reading advice.** Open read-only (`?mode=ro` or `immutable=1`), because the app may hold the
file open. Never rely on column *order* — see §2.2.

---

## 2. Schema

### 2.1 Current DDL (as created by 3.5.0)

```sql
CREATE TABLE `config` (`param` TEXT, `value` TEXT, PRIMARY KEY(param));

CREATE TABLE `cues` (
  `number`  INTEGER NOT NULL DEFAULT 999,
  `point`   INTEGER NOT NULL DEFAULT 0,
  `name`    TEXT,
  `dca01Channels` TEXT, `dca02Channels` TEXT, `dca03Channels` TEXT, `dca04Channels` TEXT,
  `dca05Channels` TEXT, `dca06Channels` TEXT, `dca07Channels` TEXT, `dca08Channels` TEXT,
  `dca01Label` TEXT, `dca02Label` TEXT, `dca03Label` TEXT, `dca04Label` TEXT,
  `dca05Label` TEXT, `dca06Label` TEXT, `dca07Label` TEXT, `dca08Label` TEXT,
  `channelPositions` TEXT,
  `channelProfiles`  TEXT,
  `fxMutes`          TEXT,
  `channelFX`        TEXT,
  `snippets`         TEXT,
  `qLabCue`          TEXT,
  `channelLevels`    TEXT,
  `scenes`           TEXT,
  `colour`           INTEGER,
  `scenePoints`      TEXT,
  `skip`             INTEGER DEFAULT 0,
  `dca09Channels` TEXT, `dca09Label` TEXT, `dca10Channels` TEXT, `dca10Label` TEXT,
  `dca11Channels` TEXT, `dca11Label` TEXT, `dca12Channels` TEXT, `dca12Label` TEXT
);
CREATE UNIQUE INDEX `cueID` ON `cues` (`number`, `point`);

CREATE TABLE `positions` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT, `name` TEXT, `shortName` TEXT,
  `delay` NUMERIC, `pan` NUMERIC, `buses` TEXT
);

CREATE TABLE `profiles` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT, `channel` INTEGER, `name` TEXT, `label` TEXT,
  `default` INTEGER DEFAULT 0, `data` TEXT
);

-- NB: the DDL really does contain the typo `TEXT`channelProfiles` TEXT` (missing comma).
-- SQLite accepts it: column `channels` gets the multi-token declared type
-- "TEXT`channelProfiles` TEXT", and the real `channelProfiles` column follows.
-- Address columns by name; tools that parse declared types may be confused.
CREATE TABLE `ensembles` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT, `name` TEXT,
  `channels` TEXT`channelProfiles` TEXT, `channelProfiles` TEXT
);

CREATE TABLE `snippetCache` (`snippet` INTEGER PRIMARY KEY, `name` TEXT);
CREATE TABLE `fxCache`      (`fx`      INTEGER PRIMARY KEY, `name` TEXT);
CREATE TABLE `sceneCache`   (`scene`   INTEGER PRIMARY KEY, `point` INTEGER DEFAULT 0, `name` TEXT);

CREATE TABLE `actors` (
  `id` INTEGER PRIMARY KEY AUTOINCREMENT, `channel` INTEGER, `name` TEXT,
  `order` INTEGER DEFAULT 0, `active` INTEGER DEFAULT 0
);
CREATE TABLE `actorProfiles` (`actor` INTEGER, `profile` INTEGER, `data` TEXT);
CREATE UNIQUE INDEX `actorProfileID` ON `actorProfiles` (`actor`, `profile`);
CREATE TABLE `actorGroups` (`id` INTEGER PRIMARY KEY AUTOINCREMENT, `name` TEXT, `data` TEXT);

CREATE TABLE sqlite_sequence(name, seq);   -- SQLite internal (AUTOINCREMENT bookkeeping)
```

### 2.2 Schema evolution — what a parser must tolerate **[C]**

The app migrates old files lazily on open, using `PRAGMA table_info(...)` +
`ALTER TABLE ... ADD COLUMN`, `SELECT type FROM sqlite_master WHERE name = ...` +
`CREATE TABLE`, and per-parameter `SELECT value FROM config WHERE param = ...` + `INSERT`.
A file on disk reflects the *newest version that saved it*, so any subset of the following may
be absent:

| Object | Added by migration (may be missing in older files) |
|---|---|
| `cues` columns | `channelFX`, `snippets`, `qLabCue`, `channelLevels`, `scenes`, `colour`, `scenePoints`, `skip`, `dca09Channels`…`dca12Label` |
| `profiles` column | `label` |
| `positions` column | `buses` |
| `ensembles` column | `channelProfiles` |
| `sceneCache` column | `point` |
| Whole tables | `snippetCache`, `fxCache`, `sceneCache`, `actors`, `actorProfiles`, `actorGroups` |
| `config` params | Any (see §4 for defaults) |

Consequences:

1. **Address columns by name, never by ordinal.** Column order differs between freshly-created
   and migrated files (e.g. `skip` precedes `dca09Channels` in new files but would follow
   `dca12Label` in a migrated one).
2. **Missing column ⇒ treat as `NULL`** for every row. Missing table ⇒ empty. Missing config
   param ⇒ its default (§4).
3. Columns added by migration are `NULL` in pre-existing rows; the app writes `''` for empty
   values on save. `NULL` and `''` are semantically identical everywhere.

Schema variants observed in the sample corpus:

| Variant | Provenance | Missing |
|---|---|---|
| A (oldest) | Files last saved by a pre-3.x-era release (SQLite 3.36 writer) | `cues.scenePoints`, `cues.skip`, `profiles.label`, `sceneCache.point`; config lacks `cueZeroActorLabels`, `cueZeroScenePoints`, `daw*`, `dimDCAFadersSuppressColours`, `selectOnSpill` |
| B | Files last saved by an intermediate 3.x release | `cues.skip`; config lacks `cueZeroActorLabels`, `daw*`, `selectOnSpill` |
| C (current) | Files saved by 3.5.0-era releases | — |

### 2.3 Version gating

`config.minVersion` (e.g. `3.0`, `3.1`) is the minimum TheatreMix version able to open the file;
the app refuses files with a higher `minVersion` than itself ("The file requires a newer version
of TheatreMix"). A parser should read it but need not enforce it. `config.profileSchemaVersion`
(§4, §5.4) versions the `profiles.data` grammar. **[C]**

---

## 3. Serialisation primitives

All structured values are stored as **TEXT** using these micro-formats. None of them supports
quoting or escaping; separators never appear inside tokens. **[C]**

| Name | Grammar | Notes | Example |
|---|---|---|---|
| **Bool** | `0` \| `1` | Config values only. | `1` |
| **Int** | `-?[0-9]+` | | `-1` |
| **IntList** | `Int ("," Int)*` \| `""` | No spaces. Order is *insertion/console* order, not guaranteed sorted — treat as a **set** unless stated. | `1,4,6,9,10` |
| **IntMap** | `Int "=" Value ("," Int "=" Value)*` \| `""` | Keys unique. Emitted from a hash table — **order is arbitrary**. | `22=5,20=5,21=5` |
| **PlusList** | `Int ("+" Int)*` | Only as a Value inside `channelFX`. | `3+4` |
| **SemiList** | `word (";" word)*` | Only inside profile `data` (`storeParams`, `actorParams`). | `gain;hpf;eq` |
| **Empty** | `NULL` or `""` | Equivalent: empty list / empty map / unset. | |

### 3.1 Implementation behind the micro-formats **[C]**

There is **no serialisation library** behind §3 — and the app is **not Python**. TheatreMix is
a native C++ application built on **Qt 5.15.14** (universal x86_64/arm64 Mach-O; links
`QtCore`, `QtSql`, `QtNetwork`, `QtWidgets`, `libc++`; no Python framework, PyQt/PySide,
shiboken or `.pyc` anywhere in the bundle). Third-party code is limited to Crypto++ (licensing),
Ross Bencina's *oscpack* (OSC) and Qt's bundled SQLite driver. RTTI names show hand-written
classes (`Show`, `Engine`, `CueList`, `X32Driver`, `DLiveDriver`, `UndoSetCueColour`, …).

The `k=v,k=v` maps are produced by the classic Qt idiom, visible in the binary as the format
string `%1=%2,` (sitting next to the `config` DDL and the `fxBusMap` defaults) with
`QString::chop(int)` imported to trim the trailing separator; plain lists are joined with
`QStringList::join` (`QtPrivate::QStringList_join` is imported) or the same `%1,` + `chop` idiom:

```cpp
QString out;
for (auto it = map.constBegin(); it != map.constEnd(); ++it)   // QHash<int,int> ⇒ arbitrary order
    out += QString("%1=%2,").arg(it.key()).arg(it.value());
out.chop(1);                                                   // drop trailing ','
```

and read back with `QString::split(QChar)` (imported) on `','` then `'='`, values via
`QString::toInt()`. The arbitrary key order of `IntMap` values is the fingerprint of iterating a
`QHash` rather than a `QMap`. `QRegularExpression::match` is imported for the input validators
quoted in this document.

Qt facilities that *are* linked but are **not** used for the show file: `QJsonDocument`/`QJsonObject`
(actor-values import/export to a standalone `… actor values.json` file), `QSettings` (application
preferences, not the show), and `QDataStream` (OSC/network float packing — `setByteOrder`,
`setFloatingPointPrecision`, `operator<<(float)` — and `QListWidgetItem` drag/drop). No
`QCbor`/`QXmlStream` at all. So a parser needs nothing beyond SQLite and string splitting.

Parser helpers (pseudo-code):

```python
def int_list(s):  return [int(t) for t in s.split(',') if t] if s else []
def int_map(s, value=int):
    out = {}
    for tok in (s or '').split(','):
        if tok:
            k, v = tok.split('=', 1)
            out[int(k)] = value(v)
    return out
```

---

## 4. `config` table — show settings **[C unless marked]**

Key/value store; `param` is the primary key. Values are TEXT even when numeric.
Defaults below are what 3.5.0 inserts for a new file or a file lacking the param.
Params marked *console-specific* are filled in from the selected console family at creation.

### 4.1 Show identity

| param | type | default | meaning |
|---|---|---|---|
| `designer` | str | `""` | Designer name. |
| `venue` | str | `""` | Venue name. |
| `minVersion` | str | app-specific (`3.0`, `3.1` seen) | Minimum app version required (§2.3). |
| `profileSchemaVersion` | int | `2` for new files (`1` inserted when migrating very old files) | Grammar version of `profiles.data` / `actorProfiles.data`. Only `2` observed. |
| `targetConsole` | str | chosen at creation | Console family the UI is configured for. Observed: `X32`, `X32C`, `M32`, `M32R`, `WING`, `WINGC`, `CDM48`, `QL5`. Full list in the binary: `X32 X32C X32P X32RACK X32CORE M32 M32R M32C WING WINGC SQ-5 SQ-6 SQ-7 TF1 TF3 TF5 QL1 QL5 CL1 CL3 CL5 GLD-80 GLD-112 Avantis Avantis-Solo DM32 DM0 DM48 DM64 CDM32 CDM48 CDM64 DM7 DM7C Qu-5 Qu-6 Qu-7 CQ…`. |

### 4.2 Last-connected console (informational)

| param | type | default | meaning |
|---|---|---|---|
| `consoleModel` | str | `""` | Model code of the last console connected (may differ from `targetConsole`, e.g. `X32C` vs `X32`). |
| `consoleVersion` | str | `""` | Console firmware version. |
| `consoleIP` | str | `""` | IPv4 address. |
| `consoleMAC` | str | `""` | MAC address. |
| `autoConnect` | bool | `0` | Auto-connect to console on open. |

### 4.3 Controlled resources

| param | type | default | meaning |
|---|---|---|---|
| `channels` | IntList | `""` | Channels under TheatreMix control, in console order. Max 48. Negative values are Aux-In channels (§5.0). Every channel here has exactly one `profiles` row with `default=1`. |
| `dcas` | IntList | `""` | Console DCA numbers under control (1-based). Length determines which `dcaNN*` columns are meaningful (max 12). |
| `backupChannels` | IntMap? | `""` | Fixed backup channel per primary channel. **[?]** Empty in every example; help describes "a backup channel assigned to each channel", so `primary=backup` is the likely shape. |
| `spareBackup` | Int | `0` | Channel number of the floating "spare backup" mic; `0` = none. |
| `fxAssigns` | IntList | `""` | TheatreMix FX bus numbers (1–4) that may be assigned to channels in cues. |
| `fxMutes` | IntList | `""` | TheatreMix FX bus numbers whose mutes are cue-programmed. Enables the FX-mute column. |
| `defaultFX` | Int | `-1` | FX bus applied to channels with no explicit FX in a cue; `-1` = none. |
| `fxBusMap` | IntMap fx→busId | `1=13,2=14,3=15,4=16` on X32/M32; console-specific elsewhere | Maps TheatreMix FX bus n to a console bus id (§5.0 "Bus ids"). An "Inhibited" mapping exists in the UI; its stored encoding was not observed **[?]**. |

### 4.4 Feature switches

| param | type | default | meaning |
|---|---|---|---|
| `channelLevels` | bool | `0` | Level-offsets feature enabled (cue column `channelLevels`). |
| `cueZeroResetLevels` | bool | `0` | Force all channel faders to 0 dB in the line-checks cue. |
| `cueZeroActorLabels` | bool | `0` | Show actor names on scribble strips in the `0.x` checks cues. |
| `snippetRecall` | bool | `0` | Console snippet recall enabled (cue column `snippets`). |
| `sceneRecall` | bool | `0` | Console scene recall enabled (cue columns `scenes`, `scenePoints`). |
| `cueZeroSnippets` | IntList | `""` | Snippets recalled by the implicit cue 0. |
| `cueZeroScenes` | IntList | `""` | Scenes recalled by cue 0. |
| `cueZeroScenePoints` | IntList | `""` | Scene points parallel to `cueZeroScenes` (§5.2 `scenePoints`). |
| `qLabCues` | Int | `0` | Playback-cue recall: `0` off, `1` QLab, `2` Show Cue System, `3` Cue Player **[I]** (ordering taken from the binary's playback list; `2` observed on a Windows/dLive show). Enables cue column `qLabCue`. |
| `qLabPasscode` | str | `""` | QLab OSC passcode. |
| `qLabSuppressBack` | bool | `0` | Don't re-fire playback cues on *Back*. |
| `dawRemote` | bool | `0` | REAPER link enabled. |
| `dawIP` | str | `""` (UI shows `127.0.0.1`) | REAPER IP. |
| `dawPreRoll` | Int | `0` | REAPER pre-roll seconds. |
| `gangLR` | bool | `0` | LR-fader ganging (X32/M32 only). |
| `gangLRChannels` | list of channel tokens | `""` | Channels ganged to the LR fader. Token grammar from the binary: `^((\d{1,2})\|(-\d)\|(Fx \d(L\|R))\|(Bus \d{1,2}))$` — input, aux-in, FX return, mix bus. **[C regex, unobserved in data]** |
| `gangLRName` | str | `""` | LR scribble-strip label. |
| `gangLRColour` | Int or `""` | `""` | Console colour index for the LR scribble (e.g. `11`). |
| `labelLR` | Int | console-specific | Show cue info on a scribble strip: `0` off, `1` on. |
| `labelTargetBus` | busId | console-specific; `1000` = LR/main | Which strip shows cue info (LR, or a non-controlled DCA — `1408` on a WING with DCAs 1–6 controlled). |

### 4.5 Console behaviour

| param | type | default | meaning |
|---|---|---|---|
| `consoleMuteDCAUnassign` | bool | `1` | While editing is unlocked, a channel's mute button unassigns it from its DCA. |
| `suppressDCAMuteBackupSwitch` | bool | `0` | Suppress DCA-mute-button backup switching. |
| `selectOnSpill` | bool | `1` | DCA-spill related option **[?]** (exact semantics not documented). |
| `enableChannelMonitoring` | bool | `1` | Automatic channel (silence/clip) monitoring. |
| `activeChannelHighlight` | bool | `0` | Yellow scribble strips on channels active in the current cue. |
| `dimDCAFaders` | bool | `0` | Dim inactive DCA scribbles to blue. |
| `dimDCAFadersSuppressColours` | bool | `0` | With the above, show active DCAs in white. |
| `qlclDyn1` | bool | `0` | Yamaha QL/CL/DM7: manage dynamics slot 1 instead of 2. |
| `buttonMap` | IntMap action→button | console-specific, e.g. `0=12,1=11` | Console assign/user button number per action. Action `0` = TheatreMix **Go**, `1` = **Back** **[C]**. Other indices observed: 2, 3, 4, 5, 6, 11. The binary's action list (UI order) is: TMix Go, TMix Back, QLab Go, QLab Stop, QLab Panic, QLab Pause, QLab Resume, SCS Go, SCS Stop, SCS Fade, SCS PauseRes, CP Go, CP Stop, CP Fade, CP Pause, CP Resume, TMix Mark, TMix RecOffset, TMix CloneOffset, REAPER Play/Pause, REAPER Stop — but the numeric index→action mapping beyond 0/1 is **[?]**. |
| `muteButtonMap` | IntMap action→muteGroup | `""` | Same action indices mapped to console mute-group buttons (1–8). **[I]** |
| `muteButtonAssignKeys` | IntMap muteGroup→softKey | `""` | On consoles whose mute groups are driven from soft keys (A&H), the soft-key number for each mute group referenced above (e.g. `8=19,6=17`). **[I]** |

---

## 5. Tables

### 5.0 Identifier domains **[C unless marked]**

| Domain | Encoding |
|---|---|
| **Channel** | Positive int = console input channel (e.g. 1–32 on X32, up to 128 on dLive — `65…88` seen). **Negative int `-n` = Aux In n** (X32/M32 aux-ins 1–6, 7/8 = USB L/R). Basis: validator `^-(\d)$`, OSC API "prefix aux ins with A", and a sampled show whose negative-numbered channels are named as a stereo playback feed rather than as performers. |
| **DCA** | 1–12, addressed via columns `dca01…dca12`. Console DCA numbering. |
| **Position id** | `positions.id`; `0` is the built-in default ("Centre Stage"). |
| **Profile id** | `profiles.id`. |
| **Ensemble id** | `ensembles.id`; `1` is the implicit "All" ensemble (never stored; the app queries `WHERE id != 1`); `2` = Male, `3` = Female are created with the file. |
| **Actor id** | `actors.id`. |
| **FX bus** | TheatreMix-side FX number, single digit (validator `^((\d)((,\|;)( )?(\d)){0,3})?$`), 1–4 observed; mapped to a console bus via `config.fxBusMap`. |
| **Bus id** | Console-specific integer. X32/M32: mix-bus number 1–16 directly. Other consoles use ≥1000 codes: `1000` = LR/Main, `1101…1104` (dLive default FX mapping), `1181`, `1183`, `1302` (a position bus), `1408` (a WING DCA). Likely `1000 + 100·type + index` **[I]**; treat as **opaque** **[?]**. |
| **Cue id** | `(number, point)` — see §5.2. |
| **Snippet / scene number** | Console indices. X32 snippets are 0-based (`0` = first slot observed). Yamaha TF: 0–99 ⇒ A00–A99, 100–199 ⇒ B00–B99. Yamaha QL/CL scene numbers `x.yy` split into `scene` + `point`. |
| **Colour index** | See §5.2 `colour`. |

### 5.1 `cues` — the cue list

One row per cue. Unique key `(number, point)`.

#### Cue numbering **[C]**

- `number`: 0–9999 (validator `^(\d{1,4})((\.|,)(\d{1,2}))?$`), `point`: 0–99.
- Display: `number` if `point == 0`, else `number "." point` — **no zero padding**. The point is a
  literal integer suffix, **not** a decimal fraction: `0.1` (point 1) and `0.10` (point 10) are
  distinct cues (both occur within a single real-world cue list). Format string in the binary: `%1%2%3` with `%2` = `.`
  and `%3` = point only when non-zero.
- Sort order is numeric on `(number, point)`: `0.9` precedes `0.10`.
- **Cue 0 (`0`,`0`) is implicit and never stored** (absent from every sampled file). It is the fixed
  "line checks" cue: all DCAs unassigned, channels at default position/FX/profile; its snippets and
  scenes come from `config.cueZero*`. Cues `0.x` *are* stored ("checks cues").
- `number` has DDL default `999` (new-row placeholder); real cues always set it.

#### Columns

| column | type | format | meaning |
|---|---|---|---|
| `number`, `point` | int | | Cue id (above). |
| `name` | text | free text | Cue text. A leading `>` indents the row in the cue list (may be repeated). No newlines observed. |
| `dcaNNChannels` (NN = 01…12) | text | IntList (set) | Channels assigned to DCA NN in this cue. Order not significant (`5,3,2,20,19,…` occurs). Empty = DCA unassigned. Any controlled channel not present in *any* DCA is muted by the app. Columns beyond `len(config.dcas)` are unused/empty. |
| `dcaNNLabel` | text | free text | Scribble-strip label for DCA NN. Empty ⇒ the app derives it (single channel: profile name/label; ensemble typed by name: the ensemble name is stored here, e.g. `Male`). May be non-empty with empty channels = **placeholder DCA**. The runtime `~` backup prefix is *not* stored. |
| `channelPositions` | text | IntMap channel→positionId | Position for channels assigned in this cue. **Only non-default entries** (position ≠ 0) are stored. |
| `channelProfiles` | text | IntMap channel→profileId | Profile for assigned channels. **Only non-default profiles** are stored; the profile's `channel` always equals the key. |
| `channelFX` | text | IntMap channel→FxSpec | Explicit FX for assigned channels. `FxSpec` = `-1` (no FX, overriding `defaultFX`) \| `n` \| `n+m[+…]` (multiple buses). Absent channel ⇒ `config.defaultFX`. |
| `fxMutes` | text | IntList | **FX buses left UNMUTED in this cue**; every other bus listed in `config.fxMutes` is muted (help: "Type in the FX buses you would like unmuted… the other FX buses will be automatically muted"). Despite the name, it is the *unmute* list. |
| `snippets` | text | IntList | Console snippets recalled after the cue's DCA data is applied. (Empty in all examples; format per help "comma separated" and by analogy with `fxMutes`.) |
| `scenes` | text | IntList | Console scenes recalled. |
| `scenePoints` | text | IntList | Decimal points for `scenes`, **parallel list, same length** (Yamaha `x.yy`; `0` elsewhere). Only single-element examples observed (`scenes=1`, `scenePoints=0`) **[I]**. |
| `qLabCue` | text | string | Playback cue number to fire (QLab/SCS/Cue Player per `config.qLabCues`). Case- and whitespace-sensitive. e.g. `M01`, `MX29`. |
| `channelLevels` | text | IntMap channel→offset | Relative fader offset in **0.1 dB units**, range `-150…+50` (−15.0…+5.0 dB, the documented limits; both extremes occur). Zero is never stored. |
| `colour` | int / NULL | enum | Cue highlight. `0`/`NULL` = none; `1` red, `2` yellow, `3` green, `4` blue, `5` purple **[I]** — `0`/`NULL` = none is certain (dominant value; column NULL in migrated files); the 1–5 order follows the binary's string table `red,yellow,green,blue,purple,none` and the OSC API's `[red\|yellow\|green\|blue\|purple]`. Observed values: 0,1,2,4,5,NULL. |
| `skip` | int | bool | `1` = cue is skipped on Go (shown in purple). `NULL` ⇒ 0. |

The app's own read query (with `%1`/`%2` expanding to the `dca09…12` columns when the console has
more than 8 DCAs):

```sql
SELECT number, point, name, dca01Channels, …, dca08Channels, [dca09..12Channels,]
       dca01Label, …, dca08Label, [dca09..12Label,]
       channelPositions, channelProfiles, channelFX, fxMutes, snippets, qLabCue,
       channelLevels, scenes, colour, scenePoints, skip
FROM cues WHERE number = :number AND point = :point
```

### 5.2 `positions` — acting-position presets

| column | type | meaning |
|---|---|---|
| `id` | int PK | `0` = built-in default `Centre Stage` / `CS`, inserted at file creation. |
| `name` | text | Display name. |
| `shortName` | text | Code shown in the cue list. Uppercase alphanumerics (app strips `[^A-Z0-9]`; auto-codes `NP`, `NP1`, …). Observed `CS`, `L15`, `RAD`. |
| `delay` | NUMERIC (int observed) | Channel delay in **ms**. |
| `pan` | NUMERIC (int observed) | Pan; negative = left, `0` = centre, positive = right. Observed −70…+70 (console pan percentage). |
| `buses` | text / NULL | IntList of console **bus ids** whose sends are unmuted when a channel is at this position (all other positions' buses are muted). e.g. `1302`. Not present in variant-A files. |

### 5.3 `profiles` — channel names and per-channel processing profiles

| column | type | meaning |
|---|---|---|
| `id` | int PK | |
| `channel` | int | Channel number (may be negative for aux-ins). |
| `name` | text | For the default profile: the **channel name** used for console scribble strips (this is where "channel names" live — there is no separate channel table). For other profiles: the profile name, shown on the strip while active. |
| `label` | text / NULL | Optional short scribble label for cramped displays. Missing in variant A. Empty in all examples. |
| `default` | int | `1` = the channel's default/base profile. **Exactly one per controlled channel** (verified across the sample corpus). |
| `data` | text / NULL | Stored processing values — grammar below. |

Scribble text validator seen in the binary (likely applied to names/labels sent to the console):
`^[A-Za-z0-9 \.:_\-,\!#$%&'()*+/<>\[\]?]*$`.

#### `data` grammar (`profileSchemaVersion = 2`) **[C for structure; key list partly I]**

```
data        := pair ("," pair)*
pair        := path "=" value | "storeParams" "=" semilist | "actorParams" "=" semilist
path        := "/" segment ("/" segment)*          -- OSC-style, relative to the channel
value       := number | token                      -- no commas, no quoting
semilist    := ptype (";" ptype)*                  -- subset of: gain hpf eq dyn (in that order)
```

Example (dLive CDM48 channel, gain+hpf+eq stored):

```
/gain/headamp=5,/gain/trim=8.9,/hpf/f=100,/hpf/on=1,
/eq/1/f=190,/eq/1/g=-9.4,/eq/1/q=0.77,/eq/1/type=PEQ, … ,/eq/4/type=PEQ,/eq/on=1,
storeParams=gain;hpf;eq
```

- `storeParams` = processing types the channel is set to *store* (the "store" check-boxes);
  `actorParams` = the same for actor storage. A profile whose types have never changed may carry
  only these keys (values are inherited from the default profile at runtime).
- Parameter paths seen in the binary: `/gain/headamp`, `/gain/trim`, `/hpf/f`, `/hpf/on`,
  `/eq/on`, `/eq/{1..6}/{f,g,q,type}` (validator `^/eq/[1-6]/(type|[fgq])$`), `/dyn/{on, thr,
  ratio, knee, mgain, att|attack, hld|hold, rel|release, pos, keysrc, mix, auto, mode, det, env,
  gain, mdl, filter/on, filter/type, filter/f}`, `/preamp/{hp,hpf,hpon,trim}`, `/delay`, and
  `/backup/{gain,hpf,eq,dyn}/…` (values captured for the channel's backup mic) **[I]**.
- EQ `type` tokens: `PEQ`, `LShv`, `HShv`, `LCut`, `HCut` (plus console-specific ones such as
  X32's `VEQ`, `BU`, `BS`).
- Values are console-native units (Hz, dB, Q, 0/1). Do not assume a fixed key set: parse
  generically into `dict[path] = str` and convert on demand.

### 5.4 `ensembles` — named channel groups

| column | type | meaning |
|---|---|---|
| `id` | int PK | `1` reserved for the implicit "All" (never stored). `2` = `Male`, `3` = `Female` inserted with empty channels at creation. |
| `name` | text | Typed into a DCA cell to merge the group. |
| `channels` | text | IntList of member channels. |
| `channelProfiles` | text / NULL | IntMap channel→profileId — profile to use for a member when the ensemble is assigned (e.g. `84=43`). |

Ensembles are only consulted at assignment time; cues store the expanded channel list.

### 5.5 `actors`, `actorProfiles`, `actorGroups` — per-performer settings

`actors`: one row per performer attached to a channel.

| column | meaning |
|---|---|
| `id` | PK |
| `channel` | Channel the actor performs on. |
| `name` | Actor name. |
| `order` | Display order within the channel. |
| `active` | `1` = currently selected actor for that channel. |

`actorProfiles`: `(actor, profile)` unique → `data` in the same grammar as `profiles.data`
(values stored per actor per profile). **[C structure / I grammar — no populated example]**

`actorGroups`: `data` is an **IntMap channel→actorId** describing a cast; loading the group sets
each channel's active actor **[I]** (basis: a sampled show with two cast groups whose maps share
the same nine chorus-channel keys and assign alternating actor ids — odd ids in one group, even in
the other — i.e. one actor per channel per cast).

Note: every example has an empty `actors` table even where `sqlite_sequence` shows actors were
once created and `actorGroups` rows remain — dangling actor ids are possible; do not treat them
as errors.

### 5.6 `snippetCache`, `fxCache`, `sceneCache` — console name caches

Read-only caches of names fetched from the console so the UI can show them offline.
`INSERT OR REPLACE` keyed by console index. Content is informational; a parser may use it to
label `cues.snippets` / `scenes` / FX numbers.

| table | key | value |
|---|---|---|
| `snippetCache` | `snippet` (console snippet index) | `name` |
| `fxCache` | `fx` (TheatreMix FX bus 1–4) | console FX slot name (e.g. `Hall`, `Plate`, `Fx 1`) |
| `sceneCache` | `scene`, `point` (Yamaha decimal, `0` otherwise; column absent in variant A) | `name` |

### 5.7 `sqlite_sequence`

SQLite's AUTOINCREMENT table. Ids are never reused, so gaps are normal (`profiles` ids 17… on a
file whose channels were renumbered, etc.). Ignore.

---

## 6. Invariants verified across the sample corpus **[C]**

A parser may assert these (they held with zero violations across roughly 800 cues in a dozen
real-world show files):

1. Every channel referenced in any `dcaNNChannels` is in `config.channels`.
2. A channel appears in **at most one** DCA per cue.
3. Every key of `channelPositions`, `channelProfiles`, `channelFX`, `channelLevels` is a channel
   assigned to some DCA **in that cue**.
4. `channelPositions` never stores position `0`; every position id exists in `positions`.
5. `channelProfiles` never stores a `default=1` profile; `profiles.channel` equals the key.
6. Every FX number in `channelFX` is `-1` or in `config.fxAssigns`.
7. `channelLevels` values are in `[-150, 50]` and never `0`.
8. Every value in `cues.fxMutes` is in `config.fxMutes`.
9. Exactly one `profiles` row with `default=1` per entry of `config.channels`.
10. No row with `number=0 AND point=0`.

---

## 7. Suggested parse procedure and object model

```
open sqlite (read-only)
assert tables {config, cues, profiles} exist
cols  = {t: names from PRAGMA table_info(t) for each table present}
cfg   = dict(SELECT param, value FROM config)  ; apply defaults from §4 for missing keys
show  = Show(
  designer, venue, target_console, min_version, profile_schema_version,
  channels   = int_list(cfg.channels),
  dcas       = int_list(cfg.dcas),
  fx         = FxConfig(assigns, mutes, default_fx, bus_map=int_map(cfg.fxBusMap)),
  flags      = … (bools),
  cue_zero   = CueZero(snippets, scenes, scene_points),
  positions  = {id: Position(...)}                 ; ensure id 0 exists
  profiles   = {id: Profile(channel, name, label, is_default, data=parse_profile_data)}
  channel_names = {p.channel: p.name for p in profiles if p.is_default}
  ensembles  = {id: Ensemble(name, channels, channel_profiles)}  ; id 1 = All (implicit)
  actors / actor_profiles / actor_groups (optional tables)
  caches     (optional tables)
  cues       = sorted by (number, point) of Cue(
      number, point, name, skip, colour,
      dcas   = {n: DcaAssign(channels=set(int_list), label) for n in 1..len(cfg.dcas)},
      positions = int_map(channelPositions), profiles = int_map(channelProfiles),
      fx        = int_map(channelFX, value=parse_fx_spec),   # -1 | [n, m, ...]
      levels_db = {ch: v/10 for ch, v in int_map(channelLevels)},
      fx_unmuted = int_list(fxMutes), snippets, scenes, scene_points, playback_cue=qLabCue)
)
```

Derived helpers worth exposing:

- `cue.display_number` → `f"{number}.{point}" if point else str(number)`.
- `cue.effective_fx(ch)` → `cue.fx.get(ch, cfg.defaultFX)`; `-1` means none.
- `cue.effective_position(ch)` → `cue.positions.get(ch, 0)`.
- `cue.effective_profile(ch)` → `cue.profiles.get(ch, default_profile_id[ch])`.
- `cue.muted_channels` → `set(cfg.channels) - union(all DCA channel sets)`.
- `channel.display_name(ch)` → `f"Aux In {-ch}"`-style for negatives if no default profile.

---

## 8. Unresolved / low-confidence items

| Item | Status | What is known |
|---|---|---|
| Bus-id encoding for non-X32 consoles (`1000`, `1101`, `1181`, `1302`, `1408`) | **[?]** | `1000` = LR; `1408` is a DCA on WING; `11xx` are dLive FX/aux sends. Likely `1000 + 100·type + index`. Treat as opaque; keep the raw int. |
| `buttonMap` / `muteButtonMap` action indices ≥ 2 | **[?]** | 0 = Go, 1 = Back certain. Candidate list in §4.5. |
| `qLabCues` values 2/3 | **[I]** | 2 observed; 3 by analogy. |
| `colour` 1–5 order | **[I]** | 0/NULL = none is certain. |
| `backupChannels` format | **[?]** | Empty in all examples; probably `primary=backup` IntMap. |
| `scenePoints` with multiple scenes | **[I]** | Parallel list assumed; only single-element data seen. |
| `fxBusMap` "Inhibited" encoding | **[?]** | Not observed. |
| `actorGroups.data` key meaning | **[I]** | channel→actor (see §5.5). |
| `actorProfiles.data` grammar | **[I]** | Assumed identical to `profiles.data` (shares `storeParams`/`actorParams` strings). |
| `selectOnSpill` semantics | **[?]** | Bool, default 1. |

---

## Appendix A — App SQL (extracted from the binary, for reference)

Creation / migration:

```sql
INSERT INTO `positions` (`id`, `name`, `shortName`, `delay`, `pan`) VALUES(0, 'Centre Stage', 'CS', 0, 0);
INSERT INTO `ensembles` (`id`, `name`, `channels`) VALUES(2, 'Male', ''),(3, 'Female', '');
INSERT INTO `cues` (`number`, `point`, `name`) VALUES(1, 0, 'New Cue');
ALTER TABLE `cues` ADD COLUMN `channelFX` TEXT;        -- … snippets, qLabCue, channelLevels, scenes,
ALTER TABLE `cues` ADD COLUMN `colour` INTEGER;        --   scenePoints, skip, dca09..12{Channels,Label}
ALTER TABLE `ensembles` ADD COLUMN `channelProfiles` TEXT;
ALTER TABLE `positions` ADD COLUMN `buses` TEXT;
ALTER TABLE `profiles`  ADD COLUMN `label` TEXT;
ALTER TABLE `sceneCache` ADD COLUMN `point` INTEGER DEFAULT 0;
UPDATE `cues` SET `channelFX` = '';
```

Row access (named placeholders are Qt bind names):

```sql
INSERT INTO cues (number, point, name) VALUES (:number, :point, :name);
UPDATE cues SET name = :name, dca01Channels = :dca01Channels, …, channelPositions = :channelPositions,
  channelProfiles = :channelProfiles, channelFX = :channelFX, fxMutes = :fxMutes, snippets = :snippets,
  qLabCue = :qLabCue, channelLevels = :channelLevels, scenes = :scenes, scenePoints = :scenePoints,
  colour = :colour, skip = :skip WHERE number = :number AND point = :point;
UPDATE cues SET number = :number, point = :point WHERE number = :oldNumber AND point = :oldPoint;
DELETE FROM cues WHERE number = :number AND point = :point;
SELECT number, point FROM cues ORDER BY number, point;
SELECT number, point FROM cues WHERE channelFX LIKE '%+%';        -- finds multi-FX cues

INSERT INTO positions (name, shortName, delay, pan, buses) VALUES (:name, :shortName, :delay, :pan, :buses);
INSERT INTO profiles (channel, name, label, `default`, data) VALUES (:channel, :name, :label, :isDefault, :data);
INSERT INTO ensembles (name, channels, channelProfiles) VALUES (:name, :channels, :channelProfiles);
SELECT id FROM ensembles WHERE id != 1 ORDER BY id;
INSERT INTO actors (channel, name, `order`, `active`) VALUES (:channel, :name, :order, :isActive);
INSERT OR REPLACE INTO `actorProfiles` (`actor`, `profile`, `data`) VALUES (:actor, :profile, :data);
INSERT INTO `actorGroups` (name, data) VALUES (:name, :data);
INSERT OR REPLACE INTO `snippetCache` (`snippet`, `name`) VALUES (:snippet, :name);
INSERT OR REPLACE INTO `fxCache` (`fx`, `name`) VALUES (:fx, :name);
INSERT OR REPLACE INTO `sceneCache` (`scene`, `point`, `name`) VALUES (:scene, :point, :name);
UPDATE config SET value = :value WHERE param = :param;
```

## Appendix B — Worked example

Row from a real X32 show file, cue text elided (`config.defaultFX=-1`, `fxAssigns=1,2`,
`fxBusMap=1=13,2=14,…`; positions 1–8 defined, `2`=`L30` pan −30, `5`=`R30` pan +30):

```
number=5  point=20  name="…"  colour=5  skip=0
dca01Channels="1"   dca02Channels="2"
dca07Channels="6,7,4,5,15,10,11,8,9"   dca08Channels="22,20,21,18,19,16,17,14,12,13"
dca01Label … dca08Label = NULL
channelPositions="22=5,20=5,21=5,18=5,19=5,16=5,17=5,6=2,7=2,4=2,5=2,14=5,15=2,12=5,13=5,10=2,11=2,8=2,9=2"
channelFX="22=1+2,20=1+2,…,6=1+2,7=1+2,4=1+2,5=1+2,2=1,1=1,14=1+2,…,9=1+2"
channelLevels=""  channelProfiles=""  fxMutes=""  snippets=""  scenes=""  qLabCue=""
```

Decoded:

- Cue **5.20**, highlighted purple (`colour=5`), not skipped.
- DCA 1 = channel 1 and DCA 2 = channel 2 (labels derived at runtime from each channel's
  default-profile name).
- DCA 7 = nine-channel ensemble `{4,5,6,7,8,9,10,11,15}` (stored unsorted — treat as a set);
  DCA 8 = ten-channel ensemble `{12,13,14,16,…,22}`. No labels stored, so the app shows its
  derived ensemble labelling (and inverted colour for multi-channel DCAs).
- Channels 1 and 2 → FX bus 1 (console mix bus 13); every ensemble channel → FX buses 1 **and** 2
  (`1+2`, mix buses 13 and 14). Nothing falls back to `defaultFX` here because every assigned
  channel has an explicit entry.
- Positions: channels 22,20,21,18,19,16,17,14,12,13 at position 5 (`R30`); 6,7,4,5,15,10,11,8,9 at
  position 2 (`L30`); channels 1 and 2 are absent from the map ⇒ default position 0 (`CS`).
- No level offsets, profile overrides, FX-mute changes, snippets, scenes or playback cue.
- Controlled channels not in any DCA (3, 23, 24, 25, 26) are muted by the app.

## Appendix C — Legacy `.x32tc` files

TheatreMix's predecessor (X32 Theatre Control) saved line-oriented `key=value` text files
(`showName=`, `fileHash=`, `versionIndex=`, `lastModified=`, `consoleIP=`, `x32tcVersion=`, …).
TheatreMix imports them and re-saves as `.tmix`. Out of scope for this spec.

## Appendix D — Real-data coverage of this specification

Which features are backed by populated real-world data (and which are documented from the
binary/help alone) — useful when deciding what to unit-test against fixtures:

**Backed by populated data in the sample corpus**

- Console families: X32/X32C, M32/M32R, WING/WINGC (target only), dLive CDM48/DM48, Yamaha QL5;
  a file with no console ever connected (empty `console*` fields).
- All three schema variants (§2.2).
- `dcas` ranging 1–6 up to 1–12, including populated `dca09…dca12` columns and placeholder DCAs
  (label without channels).
- Aux-in (negative) channel numbers; dLive input numbers up to 88; cues `0.1` and `0.10` coexisting.
- `channelPositions` (with multi-position files), `positions.buses`, `channelProfiles`
  (cue- and ensemble-level), `channelFX` including `-1` and `n+m`, `fxMutes`, `channelLevels` at both
  documented extremes (`-150`, `+50`), `colour` values 0/1/2/4/5 and `NULL`, `skip` (always 0),
  `qLabCue`, `scenes` + `scenePoints` (single-element), `snippetCache`, `fxCache`, `sceneCache`,
  `ensembles.channelProfiles`, `actorGroups`, `profiles.data` (one populated row, dLive),
  non-default profiles, `spareBackup`, `gangLRName`/`gangLRColour`, `buttonMap` with four actions,
  `muteButtonMap`, `muteButtonAssignKeys`, `qLabCues=2`, `minVersion` 3.0 and 3.1.

**Not observed in any sampled file (documented from binary strings / help only)**

- `cues.snippets`, `config.cueZeroSnippets/Scenes/ScenePoints`, `backupChannels`,
  `gangLRChannels`, `profiles.label`, populated `actors` / `actorProfiles`, multi-element
  `scenes`/`scenePoints`, `skip=1`, `colour=3`, `profileSchemaVersion=1`, an "Inhibited" `fxBusMap`
  entry, Yamaha TF / SQ / Avantis / DM7 / GLD / Qu targets.
