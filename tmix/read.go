package tmix

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// ErrNotTmix reports a SQLite database that lacks the tables every TheatreMix
// show file has (config, cues, profiles).
var ErrNotTmix = errors.New("not a TheatreMix show file")

// row is one result row with every value as a nullable string. Columns that
// do not exist in the file are simply absent from the map and read as NULL.
type row map[string]sql.NullString

func (r row) str(col string) string { return text(r[col]) }

func (r row) integer(col string) (int, error) {
	v := text(r[col])
	if v == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("column %s: bad integer %q", col, v)
	}
	return n, nil
}

func (r row) number(col string) (float64, error) {
	v := text(r[col])
	if v == "" {
		return 0, nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, fmt.Errorf("column %s: bad number %q", col, v)
	}
	return f, nil
}

func (r row) boolean(col string) (bool, error) {
	n, err := r.integer(col)
	return n != 0, err
}

// selectExisting reads the wanted columns that actually exist in table. A
// missing table yields no rows; missing columns are absent from each row.
func selectExisting(db *sql.DB, sc *schema, table string, wanted []string, orderBy string) ([]row, error) {
	if !sc.hasTable(table) {
		return nil, nil
	}
	var cols []string
	for _, c := range wanted {
		if sc.hasColumn(table, c) {
			cols = append(cols, c)
		}
	}
	if len(cols) == 0 {
		return nil, nil
	}
	quoted := make([]string, len(cols))
	for i, c := range cols {
		quoted[i] = quoteIdent(c)
	}
	q := "SELECT " + strings.Join(quoted, ", ") + " FROM " + quoteIdent(table)
	if orderBy != "" {
		q += " ORDER BY " + orderBy
	}
	rows, err := db.Query(q)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", table, err)
	}
	defer rows.Close()
	var out []row
	for rows.Next() {
		vals := make([]sql.NullString, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, fmt.Errorf("%s: %w", table, err)
		}
		r := make(row, len(cols))
		for i, c := range cols {
			r[c] = vals[i]
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", table, err)
	}
	return out, nil
}

// Read decodes an open (read-only) .tmix database into a Show.
func Read(db *sql.DB) (*Show, error) {
	sc, err := loadSchema(db)
	if err != nil {
		return nil, err
	}
	for _, t := range []string{"config", "cues", "profiles"} {
		if !sc.hasTable(t) {
			return nil, fmt.Errorf("%w: missing table %q", ErrNotTmix, t)
		}
	}
	raw, err := readRawConfig(db)
	if err != nil {
		return nil, err
	}
	cfg, info, minVersion, psv, err := buildConfig(raw)
	if err != nil {
		return nil, err
	}
	show := &Show{
		Format: Format{
			SchemaVariant:        sc.variant(),
			MinVersion:           minVersion,
			ProfileSchemaVersion: psv,
			Tables:               sc.tableNames(),
			OptionalColumns:      sc.presentOptionalColumns(),
		},
		Info:   info,
		Config: cfg,
	}
	if show.Positions, err = readPositions(db, sc); err != nil {
		return nil, err
	}
	if show.Profiles, err = readProfiles(db, sc); err != nil {
		return nil, err
	}
	if show.Ensembles, err = readEnsembles(db, sc); err != nil {
		return nil, err
	}
	if show.Actors, err = readActors(db, sc); err != nil {
		return nil, err
	}
	if show.ActorProfiles, err = readActorProfiles(db, sc); err != nil {
		return nil, err
	}
	if show.ActorGroups, err = readActorGroups(db, sc); err != nil {
		return nil, err
	}
	if show.Caches, err = readCaches(db, sc); err != nil {
		return nil, err
	}
	if show.Cues, err = readCues(db, sc, &cfg); err != nil {
		return nil, err
	}
	show.Channels = buildChannels(cfg, show.Profiles)
	return show, nil
}

func readPositions(db *sql.DB, sc *schema) ([]Position, error) {
	rows, err := selectExisting(db, sc, "positions",
		[]string{"id", "name", "shortName", "delay", "pan", "buses"}, `"id"`)
	if err != nil {
		return nil, err
	}
	out := []Position{}
	for _, r := range rows {
		var p Position
		if p.ID, err = r.integer("id"); err == nil {
			if p.DelayMs, err = r.number("delay"); err == nil {
				if p.Pan, err = r.number("pan"); err == nil {
					p.Buses, err = ParseIntList(r["buses"])
				}
			}
		}
		if err != nil {
			return nil, fmt.Errorf("positions id %s: %w", r.str("id"), err)
		}
		p.Name, p.ShortName = r.str("name"), r.str("shortName")
		out = append(out, p)
	}
	return out, nil
}

func readProfiles(db *sql.DB, sc *schema) ([]Profile, error) {
	rows, err := selectExisting(db, sc, "profiles",
		[]string{"id", "channel", "name", "label", "default", "data"}, `"id"`)
	if err != nil {
		return nil, err
	}
	out := []Profile{}
	for _, r := range rows {
		var p Profile
		if p.ID, err = r.integer("id"); err == nil {
			if p.Channel, err = r.integer("channel"); err == nil {
				if p.IsDefault, err = r.boolean("default"); err == nil {
					p.Data, err = ParseProfileData(r["data"])
				}
			}
		}
		if err != nil {
			return nil, fmt.Errorf("profiles id %s: %w", r.str("id"), err)
		}
		p.Name, p.Label = r.str("name"), r.str("label")
		out = append(out, p)
	}
	return out, nil
}

func readEnsembles(db *sql.DB, sc *schema) ([]Ensemble, error) {
	rows, err := selectExisting(db, sc, "ensembles",
		[]string{"id", "name", "channels", "channelProfiles"}, `"id"`)
	if err != nil {
		return nil, err
	}
	out := []Ensemble{}
	for _, r := range rows {
		var e Ensemble
		if e.ID, err = r.integer("id"); err == nil {
			if e.Channels, err = ParseIntList(r["channels"]); err == nil {
				e.ChannelProfiles, err = ParseIntMapInt(r["channelProfiles"])
			}
		}
		if err != nil {
			return nil, fmt.Errorf("ensembles id %s: %w", r.str("id"), err)
		}
		e.Name = r.str("name")
		out = append(out, e)
	}
	return out, nil
}

func readActors(db *sql.DB, sc *schema) ([]Actor, error) {
	rows, err := selectExisting(db, sc, "actors",
		[]string{"id", "channel", "name", "order", "active"}, `"id"`)
	if err != nil {
		return nil, err
	}
	out := []Actor{}
	for _, r := range rows {
		var a Actor
		if a.ID, err = r.integer("id"); err == nil {
			if a.Channel, err = r.integer("channel"); err == nil {
				if a.Order, err = r.integer("order"); err == nil {
					a.Active, err = r.boolean("active")
				}
			}
		}
		if err != nil {
			return nil, fmt.Errorf("actors id %s: %w", r.str("id"), err)
		}
		a.Name = r.str("name")
		out = append(out, a)
	}
	return out, nil
}

func readActorProfiles(db *sql.DB, sc *schema) ([]ActorProfile, error) {
	rows, err := selectExisting(db, sc, "actorProfiles",
		[]string{"actor", "profile", "data"}, `"actor", "profile"`)
	if err != nil {
		return nil, err
	}
	out := []ActorProfile{}
	for _, r := range rows {
		var ap ActorProfile
		if ap.Actor, err = r.integer("actor"); err == nil {
			if ap.Profile, err = r.integer("profile"); err == nil {
				ap.Data, err = ParseProfileData(r["data"])
			}
		}
		if err != nil {
			return nil, fmt.Errorf("actorProfiles actor %s profile %s: %w", r.str("actor"), r.str("profile"), err)
		}
		out = append(out, ap)
	}
	return out, nil
}

func readActorGroups(db *sql.DB, sc *schema) ([]ActorGroup, error) {
	rows, err := selectExisting(db, sc, "actorGroups", []string{"id", "name", "data"}, `"id"`)
	if err != nil {
		return nil, err
	}
	out := []ActorGroup{}
	for _, r := range rows {
		var g ActorGroup
		if g.ID, err = r.integer("id"); err == nil {
			g.ChannelActors, err = ParseIntMapInt(r["data"])
		}
		if err != nil {
			return nil, fmt.Errorf("actorGroups id %s: %w", r.str("id"), err)
		}
		g.Name = r.str("name")
		out = append(out, g)
	}
	return out, nil
}

func readCaches(db *sql.DB, sc *schema) (Caches, error) {
	c := Caches{Snippets: map[int]string{}, FX: map[int]string{}, Scenes: []SceneCacheEntry{}}
	rows, err := selectExisting(db, sc, "snippetCache", []string{"snippet", "name"}, `"snippet"`)
	if err != nil {
		return c, err
	}
	for _, r := range rows {
		k, err := r.integer("snippet")
		if err != nil {
			return c, fmt.Errorf("snippetCache: %w", err)
		}
		c.Snippets[k] = r.str("name")
	}
	if rows, err = selectExisting(db, sc, "fxCache", []string{"fx", "name"}, `"fx"`); err != nil {
		return c, err
	}
	for _, r := range rows {
		k, err := r.integer("fx")
		if err != nil {
			return c, fmt.Errorf("fxCache: %w", err)
		}
		c.FX[k] = r.str("name")
	}
	if rows, err = selectExisting(db, sc, "sceneCache", []string{"scene", "point", "name"}, `"scene"`); err != nil {
		return c, err
	}
	for _, r := range rows {
		var e SceneCacheEntry
		if e.Scene, err = r.integer("scene"); err == nil {
			e.Point, err = r.integer("point")
		}
		if err != nil {
			return c, fmt.Errorf("sceneCache: %w", err)
		}
		e.Name = r.str("name")
		c.Scenes = append(c.Scenes, e)
	}
	sort.SliceStable(c.Scenes, func(i, j int) bool {
		if c.Scenes[i].Scene != c.Scenes[j].Scene {
			return c.Scenes[i].Scene < c.Scenes[j].Scene
		}
		return c.Scenes[i].Point < c.Scenes[j].Point
	})
	return c, nil
}
