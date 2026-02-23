-- Indexes for SuwaCare LK / NCVMS

-- Users
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_nic ON users(nic);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_area_code ON users(area_code) WHERE area_code IS NOT NULL;

-- Password reset tokens
CREATE INDEX idx_password_reset_tokens_user_id ON password_reset_tokens(user_id);
CREATE INDEX idx_password_reset_tokens_expires_at ON password_reset_tokens(expires_at) WHERE used_at IS NULL;

-- Children
CREATE INDEX idx_children_registration_number ON children(registration_number);
CREATE INDEX idx_children_parent_id ON children(parent_id) WHERE parent_id IS NOT NULL;
CREATE INDEX idx_children_registered_by ON children(registered_by) WHERE registered_by IS NOT NULL;
CREATE INDEX idx_children_area_code ON children(area_code);
CREATE INDEX idx_children_created_at ON children(created_at);

-- Vaccination records
CREATE INDEX idx_vaccination_records_child_id ON vaccination_records(child_id);
CREATE INDEX idx_vaccination_records_vaccine_id ON vaccination_records(vaccine_id);
CREATE INDEX idx_vaccination_records_administered_date ON vaccination_records(administered_date);
CREATE INDEX idx_vaccination_records_status ON vaccination_records(status);

-- Vaccination schedules
CREATE INDEX idx_vaccination_schedules_child_id ON vaccination_schedules(child_id);
CREATE INDEX idx_vaccination_schedules_vaccine_id ON vaccination_schedules(vaccine_id);
CREATE INDEX idx_vaccination_schedules_due_date ON vaccination_schedules(due_date);
CREATE INDEX idx_vaccination_schedules_status ON vaccination_schedules(status);

-- Growth records
CREATE INDEX idx_growth_records_child_id ON growth_records(child_id);
CREATE INDEX idx_growth_records_recorded_date ON growth_records(recorded_date);

-- Notifications
CREATE INDEX idx_notifications_recipient_id ON notifications(recipient_id);
CREATE INDEX idx_notifications_is_read ON notifications(recipient_id, is_read);
CREATE INDEX idx_notifications_sent_date ON notifications(sent_date);

-- Reports
CREATE INDEX idx_reports_generated_by ON reports(generated_by);
CREATE INDEX idx_reports_generated_date ON reports(generated_date);
CREATE INDEX idx_reports_report_type ON reports(report_type);

-- Audit logs
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp);
CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
