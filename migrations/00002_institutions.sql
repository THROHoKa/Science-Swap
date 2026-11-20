-- +goose Up
-- +goose StatementBegin
CREATE FUNCTION public.immutable_unaccent(text) RETURNS text
LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT
AS $$ SELECT public.unaccent('public.unaccent'::regdictionary, $1) $$;
-- +goose StatementEnd

CREATE TABLE "SSwap_institutions" (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    ror_id text NOT NULL UNIQUE,
    name text NOT NULL,
    aliases text[] NOT NULL DEFAULT '{}',
    acronyms text[] NOT NULL DEFAULT '{}',
    labels jsonb NOT NULL DEFAULT '{}',
    types text[] NOT NULL,
    status text NOT NULL CHECK (status IN ('active', 'inactive', 'withdrawn')),
    domains text[] NOT NULL DEFAULT '{}',
    city text,
    subdivision_code text,
    subdivision_name text,
    country_code char(2) NOT NULL,
    continent_code char(2),
    lat double precision,
    lng double precision,
    search_text text NOT NULL,
    ror_release text NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX institutions_search_trgm ON "SSwap_institutions"
    USING gin (search_text gin_trgm_ops) WHERE status = 'active';
CREATE INDEX institutions_country ON "SSwap_institutions" (country_code);
CREATE INDEX institutions_continent ON "SSwap_institutions" (continent_code);
CREATE INDEX institutions_earth ON "SSwap_institutions"
    USING gist (ll_to_earth(lat, lng)) WHERE lat IS NOT NULL AND lng IS NOT NULL;

-- +goose Down
DROP TABLE "SSwap_institutions";
DROP FUNCTION public.immutable_unaccent(text);
