-- +goose Up
CREATE TABLE "SSwap_accounts" (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email citext NOT NULL,
    email_verified_at timestamptz,
    is_demo boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX accounts_verified_email ON "SSwap_accounts" (email)
    WHERE email_verified_at IS NOT NULL;
CREATE UNIQUE INDEX accounts_demo_email ON "SSwap_accounts" (email) WHERE is_demo;

CREATE TABLE "SSwap_sessions" (
    token_hash bytea PRIMARY KEY CHECK (octet_length(token_hash) = 32),
    account_id uuid NOT NULL REFERENCES "SSwap_accounts"(id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL
);
CREATE INDEX sessions_expiration ON "SSwap_sessions" (expires_at);

CREATE TABLE "SSwap_profiles" (
    account_id uuid PRIMARY KEY REFERENCES "SSwap_accounts"(id) ON DELETE CASCADE,
    title text NOT NULL DEFAULT '',
    name text NOT NULL CHECK (length(name) BETWEEN 1 AND 120),
    institution_id bigint NOT NULL REFERENCES "SSwap_institutions"(id),
    isced_code char(3) NOT NULL REFERENCES "SSwap_isced_fields"(code),
    teaching_areas text NOT NULL DEFAULT ''
);

CREATE TABLE "SSwap_availabilities" (
    account_id uuid PRIMARY KEY REFERENCES "SSwap_profiles"(account_id) ON DELETE CASCADE,
    period daterange NOT NULL CHECK (NOT isempty(period) AND NOT lower_inf(period) AND NOT upper_inf(period))
);
CREATE INDEX availabilities_period ON "SSwap_availabilities" USING gist (period);

CREATE TABLE "SSwap_preferences" (
    account_id uuid NOT NULL REFERENCES "SSwap_profiles"(account_id) ON DELETE CASCADE,
    scope text NOT NULL CHECK (scope IN ('world', 'continent', 'region_group', 'country', 'subdivision')),
    value text NOT NULL CHECK (length(value) <= 16 AND ((scope = 'world') = (value = ''))),
    PRIMARY KEY (account_id, scope, value)
);

CREATE TABLE "SSwap_interests" (
    from_id uuid NOT NULL REFERENCES "SSwap_profiles"(account_id) ON DELETE CASCADE,
    to_id uuid NOT NULL REFERENCES "SSwap_profiles"(account_id) ON DELETE CASCADE,
    PRIMARY KEY (from_id, to_id),
    CHECK (from_id <> to_id)
);

-- +goose StatementBegin
CREATE FUNCTION sswap_region_contains(region_scope text, region_value text, country text, continent text, subdivision text)
RETURNS boolean LANGUAGE sql STABLE AS $$
    SELECT CASE region_scope
        WHEN 'world' THEN true
        WHEN 'continent' THEN continent = region_value
        WHEN 'country' THEN country = region_value
        WHEN 'subdivision' THEN country || ':' || subdivision = region_value
        WHEN 'region_group' THEN EXISTS (
            SELECT 1 FROM "SSwap_region_groups" g
            WHERE g.code = region_value AND country = ANY(g.country_codes)
        )
        ELSE false
    END
$$;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION sswap_region_contains(text, text, text, text, text);
DROP TABLE "SSwap_interests", "SSwap_preferences", "SSwap_availabilities", "SSwap_profiles", "SSwap_sessions", "SSwap_accounts";
