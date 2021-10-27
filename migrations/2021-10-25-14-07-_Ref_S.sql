-- liquibase formatted sql

-- changeset zinov:2021-10-25-14-09-_Ref_S
CREATE TABLE IF NOT EXISTS _Ref_S
(
    id   bigserial NOT NULL,
    name varchar   NOT NULL,
    PRIMARY KEY (id)
);

INSERT INTO _Ref_S(name) VALUES('shipment1'), ('shipment1');
-- rollback drop table _Ref_S;