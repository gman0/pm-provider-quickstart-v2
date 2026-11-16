CREATE TABLE users (
    id            TEXT PRIMARY KEY,
    email         TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE contact_profiles (
    user_id       TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    full_name     TEXT,
    company       TEXT,
    support_email TEXT,
    updated_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE payment_profiles (
    user_id         TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    card_holder     TEXT,
    card_last4      TEXT,
    exp_month       INTEGER,
    exp_year        INTEGER,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE sessions (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
