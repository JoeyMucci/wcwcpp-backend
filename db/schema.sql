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
    username VARCHAR(50) UNIQUE NOT NULL
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
    contest_id UUID NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    join_code VARCHAR(8) UNIQUE NOT NULL,
    title VARCHAR(255) UNIQUE NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL
);

CREATE TABLE contest_standings (
    contest_id UUID NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_score INT NOT NULL DEFAULT 0,
    knockout_score INT NOT NULL DEFAULT 0,
    PRIMARY KEY (contest_id, user_id)
);

CREATE TABLE subcontest_entries (
    subcontest_id UUID NOT NULL REFERENCES subcontests(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (subcontest_id, user_id)
);

CREATE TABLE group_standings (
    contest_id UUID NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    country_id UUID NOT NULL REFERENCES countries(id) ON DELETE CASCADE,
    letter VARCHAR(1) NOT NULL,
    points INT NOT NULL DEFAULT 0,
    wins INT NOT NULL DEFAULT 0,
    draws INT NOT NULL DEFAULT 0,
    losses INT NOT NULL DEFAULT 0,
    gf INT NOT NULL DEFAULT 0,
    ga INT NOT NULL DEFAULT 0,
    gd INT NOT NULL DEFAULT 0,
    cs INT NOT NULL DEFAULT 0,
    rank INT,
    is_third_place_qualifier BOOLEAN,
    PRIMARY KEY (contest_id, country_id)
);

CREATE TABLE knockout_standings (
    contest_id UUID NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    country_id UUID NOT NULL REFERENCES countries(id) ON DELETE CASCADE,
    round INT NOT NULL,
    PRIMARY KEY (contest_id, country_id)
);

CREATE TABLE group_picks (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    contest_id UUID NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    country_id UUID NOT NULL REFERENCES countries(id) ON DELETE CASCADE,
    letter VARCHAR(1) NOT NULL,
    place INT NOT NULL,
    extra_qualifier BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (user_id, contest_id, country_id)
);

CREATE TABLE knockout_picks (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    contest_id UUID NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    country_id UUID NOT NULL REFERENCES countries(id) ON DELETE CASCADE,
    round INT NOT NULL,
    PRIMARY KEY (user_id, contest_id, country_id, round)
);

CREATE TABLE matches (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    contest_id UUID NOT NULL REFERENCES contests(id) ON DELETE CASCADE,
    round INT NOT NULL,
    round_index INT,
    country1_id UUID REFERENCES countries(id) ON DELETE CASCADE,
    country2_id UUID REFERENCES countries(id) ON DELETE CASCADE,
    country1_goals INT,
    country2_goals INT,
    country1_penalties INT,
    country2_penalties INT,
    country1_conduct_score INT,
    country2_conduct_score INT
);
