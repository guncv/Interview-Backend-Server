DELETE FROM issue_categories
WHERE name IN (
    'Technical Issue',
    'Account Problem',
    'Feature Request',
    'Bug Report',
    'Interview Experience',
    'Resume Issue',
    'Evaluation Concern',
    'Other'
) AND created_by IS NULL;

ALTER TABLE issue_categories
ALTER COLUMN created_by SET NOT NULL;
