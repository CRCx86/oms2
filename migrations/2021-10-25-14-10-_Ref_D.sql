-- liquibase formatted sql

-- changeset zinov:2021-10-25-14-10-_Ref_D
CREATE TABLE IF NOT EXISTS _Ref_D
(
    id   bigserial NOT NULL,
    name varchar   NOT NULL,
    PRIMARY KEY (id)
);

INSERT INTO _Ref_D(name) VALUES('delivery1'), ('delivery1');
-- rollback drop table _Ref_D;