-- +goose Up
CREATE TABLE "SSwap_region_groups" (
    code text PRIMARY KEY,
    name_en text NOT NULL,
    name_de text NOT NULL,
    country_codes char(2)[] NOT NULL CHECK (cardinality(country_codes) > 0)
);

INSERT INTO "SSwap_region_groups" (code, name_en, name_de, country_codes) VALUES
    ('EU', 'European Union', 'Europäische Union',
     '{AT,BE,BG,CY,CZ,DE,DK,EE,ES,FI,FR,GR,HR,HU,IE,IT,LT,LU,LV,MT,NL,PL,PT,RO,SE,SI,SK}'),
    ('DACH', 'DACH', 'DACH', '{DE,AT,CH}');

-- +goose Down
DROP TABLE "SSwap_region_groups";
