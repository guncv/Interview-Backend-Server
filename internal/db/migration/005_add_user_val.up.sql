DROP TABLE IF EXISTS user_roles CASCADE;

INSERT INTO users (
    email,
    password_hash,
    full_name,
    country,
    gender,
    date_of_birth,
    is_admin,
    is_email_verified
) VALUES (
    'chanagun.vir@gmail.com',
    '$2a$10$L0Z5tv0jDo3L8iuZLHW/7ugteHxc7lHm.NvFk9ft0AzeHSYAKE25e',
    'John Doe',
    'Thailand',
    'male',
    '1995-08-16',
    false,
    true
);
