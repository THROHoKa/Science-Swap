-- +goose Up
ALTER TABLE "SSwap_profiles"
    ADD COLUMN teaching_entries jsonb CHECK (jsonb_typeof(teaching_entries) = 'array');

-- +goose Down
ALTER TABLE "SSwap_profiles" DROP COLUMN teaching_entries;
