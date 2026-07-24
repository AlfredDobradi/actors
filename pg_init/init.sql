CREATE TABLE accounts (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    username VARCHAR(100) NOT NULL,
    email VARCHAR(200) NOT NULL,
    password VARCHAR(64) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    active BOOLEAN
);

CREATE TABLE sessions (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    active BOOLEAN
);

CREATE TABLE guilds (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    gold BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    UNIQUE (id, account_id)
);

CREATE TABLE heroes (
    id UUID NOT NULL,
    name VARCHAR(200) NOT NULL,
    guild_id UUID NOT NULL,
    level INT,
    cooldown INT,
    health INT,
    energy INT,
    experience INT,
    gold INT,
    last_tick TIMESTAMP DEFAULT NULL,
    status INT,
    action JSON,

    UNIQUE (id)
);

CREATE TABLE hero_resources (
    hero_id UUID NOT NULL,
    name VARCHAR(40) NOT NULL,
    amount INT,

    UNIQUE (hero_id, name)
);

CREATE TABLE hero_item (
    hero_id UUID NOT NULL,
    name VARCHAR(40) NOT NULL,
    amount INT
);