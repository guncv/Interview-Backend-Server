CREATE INDEX idx_rubrics_name_version_label_softdelete ON evaluation_rubrics (name, version_label, soft_delete);
DROP INDEX IF EXISTS idx_rubrics_name_softdelete;