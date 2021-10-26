-- liquibase formatted sql

-- changeset zinov:2021-10-25-14-07-_Ref_M
CREATE TABLE IF NOT EXISTS _Ref_M
(
    id              bigserial NOT NULL,
    name            varchar   NOT NULL,
    type            varchar NOT NULL,
    action          varchar NOT NULL,
    event_trigger   int,
    waiting_time    int,
    PRIMARY KEY (id)
);

INSERT INTO _Ref_M(name, type, action, event_trigger, waiting_time)
VALUES ('node1', 'action', 'FirstInit', null, 0),
       ('node2', 'action', 'SecondInit', null, 0),
       ('node3', 'wait', 'Wait', null, 120),
       ('node4', 'trigger', 'Trigger', 1, 0),
       ('node5', 'terminate', 'Terminate', null, 0);
-- rollback drop table _Ref_M;