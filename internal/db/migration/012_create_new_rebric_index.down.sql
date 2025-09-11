DROP INDEX IF EXISTS idx_rubrics_name_version_label_softdelete;
CREATE INDEX idx_rubrics_name_softdelete ON evaluation_rubrics (name, soft_delete);