# WHO Growth Reference Configuration

This backend supports WHO-style growth assessment using sex-specific SD curves.

## 1) Prepare official data
- Download official WHO Child Growth Standards data (0-60 months).
- Convert each month to SD points for both indicators:
  - `weight_for_age`
  - `height_for_age`
- Include male and female series.

Use `config/who_growth_reference.sample.json` as the schema template.

## 2) Configure runtime
Set environment variable:
- `WHO_GROWTH_REFERENCE_FILE` = absolute or workspace-relative path to your full JSON file.

## 3) API behavior
When WHO data is loaded:
- `ageInMonths` is still derived dynamically.
- `weightStatus` / `heightStatus` are based on WHO SD z-score rules.
- `weightZScore` / `heightZScore` are returned per record.
- `growth-records/charts` includes WHO reference series for chart overlays.

If WHO data is not loaded, the API falls back to legacy prototype thresholds.

