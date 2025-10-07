ALTER TABLE issue_categories
ALTER COLUMN created_by DROP NOT NULL;

INSERT INTO issue_categories (name, created_by)
VALUES
    ('Technical Issue', NULL),
    ('Account Problem', NULL),
    ('Feature Request', NULL),
    ('Bug Report', NULL),
    ('Interview Experience', NULL),
    ('Resume Issue', NULL),
    ('Evaluation Concern', NULL),
    ('Other', NULL)
ON CONFLICT (name) DO NOTHING;

