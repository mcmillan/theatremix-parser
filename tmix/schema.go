package tmix

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

// optionalColumns lists, per table, the columns added by app migrations that
// may be absent in older files (spec §9).
var optionalColumns = map[string][]string{
	"cues": {"channelFX", "snippets", "qLabCue", "channelLevels", "scenes", "colour",
		"scenePoints", "skip", "dca09Channels", "dca09Label", "dca10Channels", "dca10Label",
		"dca11Channels", "dca11Label", "dca12Channels", "dca12Label"},
	"profiles":   {"label"},
	"positions":  {"buses"},
	"ensembles":  {"channelProfiles"},
	"sceneCache": {"point"},
}

// schema is the set of tables and columns actually present in a file.
type schema struct {
	columns map[string]map[string]bool
}

func loadSchema(db *sql.DB) (*schema, error) {
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type = 'table' ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("sqlite_master: %w", err)
	}
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return nil, err
		}
		tables = append(tables, name)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sc := &schema{columns: map[string]map[string]bool{}}
	for _, t := range tables {
		cols, err := tableColumns(db, t)
		if err != nil {
			return nil, err
		}
		sc.columns[t] = cols
	}
	return sc, nil
}

func tableColumns(db *sql.DB, table string) (map[string]bool, error) {
	rows, err := db.Query("PRAGMA table_info(" + quoteIdent(table) + ")")
	if err != nil {
		return nil, fmt.Errorf("table_info(%s): %w", table, err)
	}
	defer rows.Close()
	cols := map[string]bool{}
	for rows.Next() {
		var (
			cid, notnull, pk int
			name, ctype      string
			dflt             sql.NullString
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return nil, fmt.Errorf("table_info(%s): %w", table, err)
		}
		cols[name] = true
	}
	return cols, rows.Err()
}

// quoteIdent double-quotes an SQL identifier.
func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func (s *schema) hasTable(t string) bool {
	_, ok := s.columns[t]
	return ok
}

func (s *schema) hasColumn(t, c string) bool {
	return s.columns[t][c]
}

func (s *schema) tableNames() []string {
	names := make([]string, 0, len(s.columns))
	for t := range s.columns {
		names = append(names, t)
	}
	sort.Strings(names)
	return names
}

// variant classifies the file per spec §9: C has cues.skip, B has
// cues.scenePoints but not skip, A has neither.
func (s *schema) variant() string {
	switch {
	case s.hasColumn("cues", "skip"):
		return "C"
	case s.hasColumn("cues", "scenePoints"):
		return "B"
	default:
		return "A"
	}
}

// presentOptionalColumns reports which migration-added columns exist, for
// each optional-bearing table that is present.
func (s *schema) presentOptionalColumns() map[string][]string {
	out := map[string][]string{}
	for t, cols := range optionalColumns {
		if !s.hasTable(t) {
			continue
		}
		present := []string{}
		for _, c := range cols {
			if s.hasColumn(t, c) {
				present = append(present, c)
			}
		}
		out[t] = present
	}
	return out
}
