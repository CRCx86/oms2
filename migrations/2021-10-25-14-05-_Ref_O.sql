-- liquibase formatted sql

-- changeset zinov:2021-10-25-14-08-_Ref_O
CREATE TABLE IF NOT EXISTS _Ref_O
(
    id   bigserial NOT NULL,
    name varchar   NOT NULL,
    PRIMARY KEY (id)
);

INSERT INTO _Ref_O(name) VALUES('order1'), ('order2');
-- rollback drop table _Ref_O;