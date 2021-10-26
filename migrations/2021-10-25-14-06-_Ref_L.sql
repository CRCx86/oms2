-- liquibase formatted sql

-- changeset zinov:2021-10-25-14-06-_Ref_L
CREATE TABLE IF NOT EXISTS _Ref_L
(
    id   bigserial NOT NULL,
    name varchar   NOT NULL,
    PRIMARY KEY (id)
    );

INSERT INTO _Ref_L(name) VALUES ('lot1'), ('lot2');
-- rollback drop table _Ref_L;