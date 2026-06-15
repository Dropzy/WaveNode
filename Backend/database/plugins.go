package database

import (
	"encoding/json"
	"fmt"
)

func (db *DB) GetPlugins(enabledOnly bool) ([]Plugin, error) {
	query := `
		SELECT id, name, version, enabled, manifest, created_at, updated_at
		FROM plugins
	`
	if enabledOnly {
		query += ` WHERE enabled = TRUE`
	}
	query += ` ORDER BY name ASC, id ASC`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugins: %v", err)
	}
	defer rows.Close()

	plugins := make([]Plugin, 0)
	for rows.Next() {
		var plugin Plugin
		if err := rows.Scan(
			&plugin.ID,
			&plugin.Name,
			&plugin.Version,
			&plugin.Enabled,
			&plugin.Manifest,
			&plugin.CreatedAt,
			&plugin.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to read plugin: %v", err)
		}
		plugins = append(plugins, plugin)
	}
	return plugins, rows.Err()
}

func (db *DB) UpsertPlugin(plugin Plugin) error {
	if !json.Valid(plugin.Manifest) {
		return fmt.Errorf("plugin manifest is not valid JSON")
	}
	_, err := db.conn.Exec(`
		INSERT INTO plugins (id, name, version, enabled, manifest, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			version = EXCLUDED.version,
			enabled = EXCLUDED.enabled,
			manifest = EXCLUDED.manifest,
			updated_at = CURRENT_TIMESTAMP
	`, plugin.ID, plugin.Name, plugin.Version, plugin.Enabled, plugin.Manifest)
	if err != nil {
		return fmt.Errorf("failed to save plugin: %v", err)
	}
	return nil
}

func (db *DB) SetPluginEnabled(id string, enabled bool) error {
	result, err := db.conn.Exec(`
		UPDATE plugins
		SET enabled = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`, id, enabled)
	if err != nil {
		return fmt.Errorf("failed to update plugin: %v", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to confirm plugin update: %v", err)
	}
	if affected == 0 {
		return fmt.Errorf("plugin not found")
	}
	return nil
}

func (db *DB) DeletePlugin(id string) error {
	result, err := db.conn.Exec(`DELETE FROM plugins WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to remove plugin: %v", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to confirm plugin removal: %v", err)
	}
	if affected == 0 {
		return fmt.Errorf("plugin not found")
	}
	return nil
}
