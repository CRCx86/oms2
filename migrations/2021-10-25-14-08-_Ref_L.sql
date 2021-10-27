-- liquibase formatted sql

-- changeset zinov:2021-10-25-14-06-_Ref_L
CREATE TABLE IF NOT EXISTS _Ref_L
(
    id   bigserial NOT NULL,
    name varchar   NOT NULL,
    order_id int NOT NULL ,
    foreign key(order_id) references _Ref_O(id) on delete cascade,
    PRIMARY KEY (id)
    );

INSERT INTO _Ref_L(name, order_id) VALUES ('lot1', 1), ('lot2', 2);
-- rollback drop table _Ref_L;