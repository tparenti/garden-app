package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/alexr/garden-app/internal/models"
)

func (s *SQLiteStore) AddPlantSpec(ctx context.Context, spec *models.PlantSpec) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO plant_specs
			(name, variety, days_to_germination, days_to_maturity, spacing_inches, depth_inches,
			 sun_requirement, water_requirement, weeks_before_frost, weeks_after_frost,
			 start_indoors, direct_sow, hardiness_zone_min, hardiness_zone_max, notes)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		spec.Name, spec.Variety, spec.DaysToGermination, spec.DaysToMaturity,
		spec.SpacingInches, spec.DepthInches, spec.SunRequirement, spec.WaterRequirement,
		spec.WeeksBeforeFrost, spec.WeeksAfterFrost, boolToInt(spec.StartIndoors),
		boolToInt(spec.DirectSow), spec.HardinessZoneMin, spec.HardinessZoneMax, spec.Notes,
	)
	if err != nil {
		return 0, fmt.Errorf("insert plant spec: %w", err)
	}
	return res.LastInsertId()
}

func (s *SQLiteStore) ListPlantSpecs(ctx context.Context) ([]models.PlantSpec, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, variety, days_to_germination, days_to_maturity, spacing_inches,
		       depth_inches, sun_requirement, water_requirement, weeks_before_frost,
		       weeks_after_frost, start_indoors, direct_sow, hardiness_zone_min,
		       hardiness_zone_max, notes
		FROM plant_specs ORDER BY name, variety`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPlantSpecs(rows)
}

func (s *SQLiteStore) GetPlantSpec(ctx context.Context, id int64) (*models.PlantSpec, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, variety, days_to_germination, days_to_maturity, spacing_inches,
		       depth_inches, sun_requirement, water_requirement, weeks_before_frost,
		       weeks_after_frost, start_indoors, direct_sow, hardiness_zone_min,
		       hardiness_zone_max, notes
		FROM plant_specs WHERE id = ?`, id)
	spec := &models.PlantSpec{}
	var startIndoors, directSow int
	err := row.Scan(&spec.ID, &spec.Name, &spec.Variety, &spec.DaysToGermination,
		&spec.DaysToMaturity, &spec.SpacingInches, &spec.DepthInches, &spec.SunRequirement,
		&spec.WaterRequirement, &spec.WeeksBeforeFrost, &spec.WeeksAfterFrost,
		&startIndoors, &directSow, &spec.HardinessZoneMin, &spec.HardinessZoneMax, &spec.Notes)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("plant spec %d not found", id)
	}
	spec.StartIndoors = startIndoors == 1
	spec.DirectSow = directSow == 1
	return spec, err
}

func (s *SQLiteStore) SearchPlantSpecs(ctx context.Context, query string) ([]models.PlantSpec, error) {
	like := "%" + query + "%"
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, variety, days_to_germination, days_to_maturity, spacing_inches,
		       depth_inches, sun_requirement, water_requirement, weeks_before_frost,
		       weeks_after_frost, start_indoors, direct_sow, hardiness_zone_min,
		       hardiness_zone_max, notes
		FROM plant_specs
		WHERE name LIKE ? OR variety LIKE ? OR notes LIKE ?
		ORDER BY name, variety`, like, like, like)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPlantSpecs(rows)
}

func scanPlantSpecs(rows *sql.Rows) ([]models.PlantSpec, error) {
	var specs []models.PlantSpec
	for rows.Next() {
		var s models.PlantSpec
		var startIndoors, directSow int
		if err := rows.Scan(&s.ID, &s.Name, &s.Variety, &s.DaysToGermination,
			&s.DaysToMaturity, &s.SpacingInches, &s.DepthInches, &s.SunRequirement,
			&s.WaterRequirement, &s.WeeksBeforeFrost, &s.WeeksAfterFrost,
			&startIndoors, &directSow, &s.HardinessZoneMin, &s.HardinessZoneMax, &s.Notes); err != nil {
			return nil, err
		}
		s.StartIndoors = startIndoors == 1
		s.DirectSow = directSow == 1
		specs = append(specs, s)
	}
	return specs, rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
