-- Migration: Client IP and user agent on operator access logs.
-- Depends on: 018_operator_iam_evidence

ALTER TABLE operator_access_logs
    ADD COLUMN ip_address TEXT NOT NULL DEFAULT '',
    ADD COLUMN user_agent TEXT NOT NULL DEFAULT '';
