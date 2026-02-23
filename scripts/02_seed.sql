-- Seed data: vaccines (Sri Lanka EPI schedule – representative set)
-- Run after 00_schema.sql and 01_indexes.sql

INSERT INTO vaccines (id, name, manufacturer, dosage_info, recommended_age, interval_days, description, is_active) VALUES
('vaccine-001', 'BCG', 'Serum Institute', '0.05ml', 0, 0, 'Bacillus Calmette-Guérin vaccine', true),
('vaccine-002', 'OPV', 'WHO', '2 drops', 60, 60, 'Oral Polio Vaccine', true),
('vaccine-003', 'Pentavalent', 'Serum Institute', '0.5ml', 60, 60, 'DPT-HepB-Hib', true),
('vaccine-004', 'PCV', 'Pfizer', '0.5ml', 60, 60, 'Pneumococcal Conjugate Vaccine', true),
('vaccine-005', 'MR', 'WHO', '0.5ml', 270, 0, 'Measles-Rubella', true),
('vaccine-006', 'JE', 'Chengdu', '0.5ml', 270, 0, 'Japanese Encephalitis', true),
('vaccine-007', 'OPV Booster', 'WHO', '2 drops', 1095, 0, 'OPV Booster', true),
('vaccine-008', 'DPT Booster', 'Serum Institute', '0.5ml', 1095, 0, 'DPT Booster', true);

-- Optional: sample users (passwords are placeholders – use proper hashes in production)
-- bcrypt hash for "parent123" (cost 10): $2a$10$...
-- Uncomment and replace password_hash with real bcrypt before use.
/*
INSERT INTO users (id, email, nic, password_hash, role, name, phone_number, address, language_preference, area_code) VALUES
('user-parent-001', 'parent@ncvms.gov.lk', '987654321V', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'parent', 'Amara Perera', '+94771234567', '123 Main Street, Colombo', 'en', NULL),
('phm-001', 'phm@ncvms.gov.lk', '123456789V', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'phm', 'Dr. Perera', '+94771234568', 'MOH Colombo', 'en', 'COL-01'),
('user-moh-001', 'moh@ncvms.gov.lk', '111222333V', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'moh', 'MOH Director', '+94771234569', 'MOH Office', 'en', NULL);
*/
