-- liquibase formatted sql

-- changeset zinov:2021-10-24-14-53-_Ref_ET
CREATE TABLE IF NOT EXISTS _Ref_ET
(
    id   bigserial  NOT NULL,
    name varchar NOT NULL,
    PRIMARY KEY (id)
);

INSERT INTO _Ref_ET(name) VALUES('event_type1'), ('event_type2');
-- rollback drop table _Ref_ET;