-- To easily drop and rebuild this schema, we drop tables in reverse dependency order
DROP TABLE IF EXISTS group_picks;
DROP TABLE IF EXISTS knockout_picks;
DROP TABLE IF EXISTS group_standings;
DROP TABLE IF EXISTS knockout_standings;
DROP TABLE IF EXISTS subcontest_entries;
DROP TABLE IF EXISTS contest_standings;
DROP TABLE IF EXISTS subcontests;
DROP TABLE IF EXISTS matches;
DROP TABLE IF EXISTS countries;
DROP TABLE IF EXISTS contests;
DROP TABLE IF EXISTS users;

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(50) UNIQUE NOT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE contests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(255) UNIQUE NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    group_unlock_date TIMESTAMP WITH TIME ZONE NOT NULL,
    group_lock_date TIMESTAMP WITH TIME ZONE NOT NULL,
    knockout_unlock_date TIMESTAMP WITH TIME ZONE NOT NULL,
    knockout_lock_date TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE countries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    code VARCHAR(3) UNIQUE NOT NULL,
    full_name VARCHAR(255) UNIQUE NOT NULL
);

CREATE TABLE subcontests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    contest_id UUID NOT NULL REFERENCES contests(id),
    user_id UUID NOT NULL REFERENCES users(id),
    join_code VARCHAR(8) UNIQUE NOT NULL,
    title VARCHAR(255) UNIQUE NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE TABLE contest_standings (
    contest_id UUID NOT NULL REFERENCES contests(id),
    user_id UUID NOT NULL REFERENCES users(id),
    score INT NOT NULL,
    PRIMARY KEY (contest_id, user_id)
);

CREATE TABLE subcontest_entries (
    subcontest_id UUID NOT NULL REFERENCES subcontests(id),
    user_id UUID NOT NULL REFERENCES users(id),
    PRIMARY KEY (subcontest_id, user_id)
);

CREATE TABLE group_standings (
    contest_id UUID NOT NULL REFERENCES contests(id),
    country_id UUID NOT NULL REFERENCES countries(id),
    letter VARCHAR(1) NOT NULL,
    points INT NOT NULL DEFAULT 0,
    PRIMARY KEY (contest_id, country_id)
);

CREATE TABLE knockout_standings (
    contest_id UUID NOT NULL REFERENCES contests(id),
    country_id UUID NOT NULL REFERENCES countries(id),
    round INT NOT NULL,
    PRIMARY KEY (contest_id, country_id)
);

CREATE TABLE group_picks (
    user_id UUID NOT NULL REFERENCES users(id),
    contest_id UUID NOT NULL REFERENCES contests(id),
    country_id UUID NOT NULL REFERENCES countries(id),
    place INT NOT NULL,
    PRIMARY KEY (user_id, contest_id, country_id)
);

CREATE TABLE knockout_picks (
    user_id UUID NOT NULL REFERENCES users(id),
    contest_id UUID NOT NULL REFERENCES contests(id),
    country_id UUID NOT NULL REFERENCES countries(id),
    round INT NOT NULL,
    PRIMARY KEY (user_id, contest_id, country_id)
);

CREATE TABLE matches (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    contest_id UUID NOT NULL REFERENCES contests(id),
    country1_id UUID NOT NULL REFERENCES countries(id),
    country2_id UUID NOT NULL REFERENCES countries(id),
    country1_goals INT,
    country2_goals INT,
    country1_penalties INT,
    country2_penalties INT
);
