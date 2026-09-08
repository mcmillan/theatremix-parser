// Package fixture builds synthetic .tmix databases for tests. No real show
// files ship with the repository, so tests construct their own from the DDL in
// docs/TMIX_FORMAT_SPEC.agent.md and then downgrade them to the older schema
// variants described there.
package fixture

import (
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	_ "modernc.org/sqlite" // registers the "sqlite" driver
)

// ddl is the current (variant C) schema, identifiers double-quoted. The
// ensembles statement reproduces the app's missing-comma typo on purpose.
var ddl = []string{
	`CREATE TABLE "config" ("param" TEXT, "value" TEXT, PRIMARY KEY(param))`,
	`CREATE TABLE "cues" (
	  "number" INTEGER NOT NULL DEFAULT 999, "point" INTEGER NOT NULL DEFAULT 0, "name" TEXT,
	  "dca01Channels" TEXT, "dca02Channels" TEXT, "dca03Channels" TEXT, "dca04Channels" TEXT,
	  "dca05Channels" TEXT, "dca06Channels" TEXT, "dca07Channels" TEXT, "dca08Channels" TEXT,
	  "dca01Label" TEXT, "dca02Label" TEXT, "dca03Label" TEXT, "dca04Label" TEXT,
	  "dca05Label" TEXT, "dca06Label" TEXT, "dca07Label" TEXT, "dca08Label" TEXT,
	  "channelPositions" TEXT, "channelProfiles" TEXT, "fxMutes" TEXT, "channelFX" TEXT,
	  "snippets" TEXT, "qLabCue" TEXT, "channelLevels" TEXT, "scenes" TEXT, "colour" INTEGER,
	  "scenePoints" TEXT, "skip" INTEGER DEFAULT 0,
	  "dca09Channels" TEXT, "dca09Label" TEXT, "dca10Channels" TEXT, "dca10Label" TEXT,
	  "dca11Channels" TEXT, "dca11Label" TEXT, "dca12Channels" TEXT, "dca12Label" TEXT)`,
	`CREATE UNIQUE INDEX "cueID" ON "cues" ("number", "point")`,
	`CREATE TABLE "positions" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "name" TEXT, "shortName" TEXT,
	  "delay" NUMERIC, "pan" NUMERIC, "buses" TEXT)`,
	`CREATE TABLE "profiles" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "channel" INTEGER, "name" TEXT,
	  "label" TEXT, "default" INTEGER DEFAULT 0, "data" TEXT)`,
	`CREATE TABLE "ensembles" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "name" TEXT,
	  "channels" TEXT"channelProfiles" TEXT, "channelProfiles" TEXT)`,
	`CREATE TABLE "snippetCache" ("snippet" INTEGER PRIMARY KEY, "name" TEXT)`,
	`CREATE TABLE "fxCache" ("fx" INTEGER PRIMARY KEY, "name" TEXT)`,
	`CREATE TABLE "sceneCache" ("scene" INTEGER PRIMARY KEY, "point" INTEGER DEFAULT 0, "name" TEXT)`,
	`CREATE TABLE "actors" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "channel" INTEGER, "name" TEXT,
	  "order" INTEGER DEFAULT 0, "active" INTEGER DEFAULT 0)`,
	`CREATE TABLE "actorProfiles" ("actor" INTEGER, "profile" INTEGER, "data" TEXT)`,
	`CREATE UNIQUE INDEX "actorProfileID" ON "actorProfiles" ("actor", "profile")`,
	`CREATE TABLE "actorGroups" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "name" TEXT, "data" TEXT)`,
}

// Config is the variant-C config table. Channels include two aux-ins; ten
// DCAs are controlled so dca09/dca10 columns are live.
var Config = map[string]string{
	"designer": "Test Designer", "venue": "Test Venue", "targetConsole": "X32",
	"consoleModel": "X32C", "consoleVersion": "4.13", "consoleIP": "192.0.2.10", "consoleMAC": "00:00:5e:00:53:01",
	"minVersion": "3.1", "profileSchemaVersion": "2", "autoConnect": "0",
	"channels": "1,2,3,4,5,6,7,8,-1,-2", "dcas": "1,2,3,4,5,6,7,8,9,10",
	"fxAssigns": "1,2", "fxMutes": "1,2,3", "defaultFX": "-1", "fxBusMap": "1=13,2=14,3=15,4=16",
	"channelLevels": "1", "snippetRecall": "1", "sceneRecall": "1",
	"qLabCues": "1", "qLabPasscode": "", "qLabSuppressBack": "1",
	"buttonMap": "1=11,0=12", "muteButtonMap": "0=8,1=6", "muteButtonAssignKeys": "8=19",
	"labelLR": "0", "labelTargetBus": "1000", "spareBackup": "8", "backupChannels": "",
	"gangLR": "0", "gangLRChannels": "", "gangLRName": "Band", "gangLRColour": "11",
	"cueZeroSnippets": "", "cueZeroScenes": "1", "cueZeroScenePoints": "0",
	"consoleMuteDCAUnassign": "1", "enableChannelMonitoring": "1", "selectOnSpill": "0",
	"dawRemote": "0", "dawIP": "127.0.0.1", "dawPreRoll": "0",
	"cueZeroActorLabels": "0", "cueZeroResetLevels": "1", "activeChannelHighlight": "0",
	"dimDCAFaders": "0", "dimDCAFadersSuppressColours": "0", "suppressDCAMuteBackupSwitch": "0", "qlclDyn1": "0",
}

type stmt struct {
	sql  string
	args []any
}

// rows are the seed rows shared by every variant. Cue rows exercise every
// micro-format rule: unordered lists/maps, n+m and -1 FX, level extremes,
// placeholder DCA, dca09/10, NULL vs "" in the same column, and 0.1 vs 0.10.
var rows = []stmt{
	{`INSERT INTO positions (id, name, shortName, delay, pan, buses) VALUES (?,?,?,?,?,?)`, []any{0, "Centre Stage", "CS", 0, 0, ""}},
	{`INSERT INTO positions (id, name, shortName, delay, pan, buses) VALUES (?,?,?,?,?,?)`, []any{1, "Left", "L", 0, -30, nil}},
	{`INSERT INTO positions (id, name, shortName, delay, pan, buses) VALUES (?,?,?,?,?,?)`, []any{2, "Radio", "RAD", 5, 0, "1302"}},

	{`INSERT INTO profiles (id, channel, name, label, "default", data) VALUES (?,?,?,?,?,?)`, []any{1, 1, "Ch1", "", 1, ""}},
	{`INSERT INTO profiles (id, channel, name, label, "default", data) VALUES (?,?,?,?,?,?)`, []any{2, 2, "Ch2", "", 1, ""}},
	{`INSERT INTO profiles (id, channel, name, label, "default", data) VALUES (?,?,?,?,?,?)`, []any{3, 3, "Ch3", "C3", 1, nil}},
	{`INSERT INTO profiles (id, channel, name, label, "default", data) VALUES (?,?,?,?,?,?)`, []any{4, 4, "Ch4", "", 1, ""}},
	{`INSERT INTO profiles (id, channel, name, label, "default", data) VALUES (?,?,?,?,?,?)`, []any{5, 5, "Ch5", "", 1, ""}},
	{`INSERT INTO profiles (id, channel, name, label, "default", data) VALUES (?,?,?,?,?,?)`, []any{6, 6, "Ch6", "", 1, ""}},
	{`INSERT INTO profiles (id, channel, name, label, "default", data) VALUES (?,?,?,?,?,?)`, []any{7, 7, "Ch7", "", 1, ""}},
	{`INSERT INTO profiles (id, channel, name, label, "default", data) VALUES (?,?,?,?,?,?)`, []any{8, 8, "Ch8", "", 1, ""}},
	{`INSERT INTO profiles (id, channel, name, label, "default", data) VALUES (?,?,?,?,?,?)`, []any{9, -1, "Aux1", "", 1, ""}},
	{`INSERT INTO profiles (id, channel, name, label, "default", data) VALUES (?,?,?,?,?,?)`, []any{10, -2, "Aux2", "", 1, ""}},
	{`INSERT INTO profiles (id, channel, name, label, "default", data) VALUES (?,?,?,?,?,?)`, []any{11, 3, "Alt3", "", 0,
		"/gain/headamp=5,/gain/trim=8.9,/hpf/f=100,/hpf/on=1,/eq/1/f=190,/eq/1/g=-9.4,/eq/1/q=0.77,/eq/1/type=PEQ,/eq/on=1,storeParams=gain;hpf;eq"}},
	{`INSERT INTO profiles (id, channel, name, label, "default", data) VALUES (?,?,?,?,?,?)`, []any{12, 4, "Alt4", "", 0, nil}},

	{`INSERT INTO ensembles (id, name, channels, channelProfiles) VALUES (?,?,?,?)`, []any{2, "Male", "1,2,3", nil}},
	{`INSERT INTO ensembles (id, name, channels, channelProfiles) VALUES (?,?,?,?)`, []any{3, "Female", "4,5,6", "4=12"}},
	{`INSERT INTO ensembles (id, name, channels, channelProfiles) VALUES (?,?,?,?)`, []any{4, "Kids", "8,7", ""}},

	{`INSERT INTO actors (id, channel, name, "order", active) VALUES (?,?,?,?,?)`, []any{1, 3, "Alice", 0, 1}},
	{`INSERT INTO actors (id, channel, name, "order", active) VALUES (?,?,?,?,?)`, []any{2, 3, "Bob", 1, 0}},
	{`INSERT INTO actorProfiles (actor, profile, data) VALUES (?,?,?)`, []any{1, 11, "/gain/trim=2,actorParams=gain"}},
	{`INSERT INTO actorGroups (id, name, data) VALUES (?,?,?)`, []any{1, "Act I", "3=1,4=99"}},

	{`INSERT INTO snippetCache (snippet, name) VALUES (?,?)`, []any{0, "Pit Mute"}},
	{`INSERT INTO fxCache (fx, name) VALUES (?,?)`, []any{1, "Hall"}},
	{`INSERT INTO sceneCache (scene, point, name) VALUES (?,?,?)`, []any{2, 50, "Half"}},
	{`INSERT INTO sceneCache (scene, point, name) VALUES (?,?,?)`, []any{1, 0, "Init"}},

	{`INSERT INTO cues (number, point, name, dca01Channels) VALUES (?,?,?,?)`, []any{0, 10, "check ten", "2"}},
	{`INSERT INTO cues (number, point, name, dca01Channels) VALUES (?,?,?,?)`, []any{0, 1, "check one", "1"}},
	{`INSERT INTO cues (number, point, name, dca01Channels, dca02Channels, dca03Channels, dca03Label,
	   channelPositions, channelProfiles, channelFX, channelLevels, fxMutes, snippets, scenes, scenePoints,
	   qLabCue, colour, skip) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		[]any{1, 0, "Scene 1", "1", "2", "5,3,4", "Trio",
			"4=2,3=1", "3=11", "5=-1,3=1+2,1=1", "4=50,3=-150", "3,1", "0", "2", "50",
			"M01", 5, 0}},
	{`INSERT INTO cues (number, point, name, dca01Channels) VALUES (?,?,?,?)`, []any{1, 9, "nine", "6"}},
	{`INSERT INTO cues (number, point, name, dca01Channels) VALUES (?,?,?,?)`, []any{1, 10, "ten", "7"}},
	{`INSERT INTO cues (number, point, name, dca09Label, dca10Channels, dca10Label, colour, skip, channelFX) VALUES (?,?,?,?,?,?,?,?,?)`,
		[]any{2, 0, "placeholder", "Bunsen", "-2,-1", "Aux", nil, 1, nil}},
	{`INSERT INTO cues (number, point, name, dca01Channels, dca02Channels, dca01Label, dca02Label) VALUES (?,?,?,?,?,?,?)`,
		[]any{3, 0, "", nil, "", nil, ""}},
}

// downgrades turn a variant-C database into an older variant (spec §9).
var downgrades = map[string][]string{
	"B": {
		`ALTER TABLE cues DROP COLUMN skip`,
		`DELETE FROM config WHERE param IN ('cueZeroActorLabels','dawRemote','dawIP','dawPreRoll','selectOnSpill')`,
	},
	"A": {
		`ALTER TABLE cues DROP COLUMN skip`,
		`ALTER TABLE cues DROP COLUMN scenePoints`,
		`ALTER TABLE profiles DROP COLUMN label`,
		`ALTER TABLE sceneCache DROP COLUMN point`,
		`DROP TABLE actors`, `DROP TABLE actorProfiles`, `DROP TABLE actorGroups`,
		`DROP TABLE snippetCache`, `DROP TABLE fxCache`,
		`DELETE FROM config WHERE param IN ('cueZeroActorLabels','dawRemote','dawIP','dawPreRoll','selectOnSpill',
		  'cueZeroScenePoints','dimDCAFadersSuppressColours')`,
	},
}

// DSN builds a read-write file: URI for path, percent-encoding it.
func DSN(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	return (&url.URL{Scheme: "file", Path: abs}).String()
}

// Create writes a fixture of the given schema variant ("A", "B" or "C") to
// path and returns path.
func Create(t testing.TB, path, variant string) string {
	t.Helper()
	if _, ok := downgrades[variant]; !ok && variant != "C" {
		t.Fatalf("unknown fixture variant %q", variant)
	}
	db, err := sql.Open("sqlite", DSN(path))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, s := range ddl {
		mustExec(t, db, s)
	}
	params := make([]string, 0, len(Config))
	for k := range Config {
		params = append(params, k)
	}
	sort.Strings(params)
	for _, k := range params {
		mustExec(t, db, `INSERT INTO config (param, value) VALUES (?,?)`, k, Config[k])
	}
	for _, r := range rows {
		mustExec(t, db, r.sql, r.args...)
	}
	for _, s := range downgrades[variant] {
		mustExec(t, db, s)
	}
	return path
}

// Mutate runs statements against an existing fixture (to break invariants).
func Mutate(t testing.TB, path string, stmts ...string) {
	t.Helper()
	db, err := sql.Open("sqlite", DSN(path))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, s := range stmts {
		mustExec(t, db, s)
	}
}

// CreateBare writes a SQLite database containing only the given DDL
// statements (for "not a tmix" tests).
func CreateBare(t testing.TB, path string, stmts ...string) string {
	t.Helper()
	db, err := sql.Open("sqlite", DSN(path))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, s := range stmts {
		mustExec(t, db, s)
	}
	return path
}

func mustExec(t testing.TB, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.Exec(query, args...); err != nil {
		t.Fatalf("exec %s: %v", firstLine(query), err)
	}
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i] + "…"
	}
	return fmt.Sprint(s)
}
