-- +goose Up
CREATE TABLE "SSwap_isced_fields" (
    code char(3) PRIMARY KEY CHECK (code ~ '^[0-9]{3}$'),
    broad_code char(2) NOT NULL CHECK (broad_code = left(code, 2)),
    name_en text NOT NULL,
    name_de text NOT NULL
);

INSERT INTO "SSwap_isced_fields" (code, broad_code, name_en, name_de) VALUES
    ('011','01','Education','Erziehungswissenschaften'),
    ('021','02','Arts','Künste'),
    ('022','02','Humanities (except languages)','Geisteswissenschaften (ohne Sprachen)'),
    ('023','02','Languages','Sprachen'),
    ('031','03','Social and behavioural sciences','Sozial- und Verhaltenswissenschaften'),
    ('032','03','Journalism and information','Journalismus und Informationswesen'),
    ('041','04','Business and administration','Wirtschaft und Verwaltung'),
    ('042','04','Law','Recht'),
    ('051','05','Biological and related sciences','Biologie und verwandte Wissenschaften'),
    ('052','05','Environment','Umwelt'),
    ('053','05','Physical sciences','Exakte Naturwissenschaften'),
    ('054','05','Mathematics and statistics','Mathematik und Statistik'),
    ('061','06','Information and Communication Technologies (ICTs)','Informations- und Kommunikationstechnologien'),
    ('071','07','Engineering and engineering trades','Ingenieurwesen und technische Berufe'),
    ('072','07','Manufacturing and processing','Verarbeitendes Gewerbe und Bergbau'),
    ('073','07','Architecture and construction','Architektur und Baugewerbe'),
    ('081','08','Agriculture','Landwirtschaft'),
    ('082','08','Forestry','Forstwirtschaft'),
    ('083','08','Fisheries','Fischerei'),
    ('084','08','Veterinary','Tiermedizin'),
    ('091','09','Health','Gesundheit'),
    ('092','09','Welfare','Sozialwesen'),
    ('101','10','Personal services','Persönliche Dienstleistungen'),
    ('103','10','Security services','Sicherheitsdienstleistungen'),
    ('104','10','Transport services','Verkehrsdienstleistungen');

-- +goose Down
DROP TABLE "SSwap_isced_fields";
