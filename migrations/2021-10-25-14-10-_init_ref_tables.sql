-- liquibase formatted sql

-- changeset zinov:2021-10-24-14-53-_Ref_ET
CREATE TABLE IF NOT EXISTS _Ref_ET
(
    id   bigserial NOT NULL,
    name varchar   NOT NULL,
    PRIMARY KEY (id)
);

INSERT INTO _Ref_ET(name)
VALUES ('event_type1'),
       ('event_type2');
-- rollback drop table _Ref_ET;

-- changeset zinov:2021-10-25-14-08-_Ref_O
CREATE TABLE IF NOT EXISTS _Ref_O
(
    id   bigserial NOT NULL,
    name varchar   NOT NULL,
    PRIMARY KEY (id)
);

INSERT INTO _Ref_O(name)
VALUES ('order1'),
       ('order2');
-- rollback drop table _Ref_O;

-- changeset zinov:2021-10-25-14-10-_Ref_D
CREATE TABLE IF NOT EXISTS _Ref_D
(
    id   bigserial NOT NULL,
    name varchar   NOT NULL,
    PRIMARY KEY (id)
);

INSERT INTO _Ref_D(name)
VALUES ('delivery1'),
       ('delivery1');
-- rollback drop table _Ref_D;

-- changeset zinov:2021-10-25-14-09-_Ref_S
CREATE TABLE IF NOT EXISTS _Ref_S
(
    id   bigserial NOT NULL,
    name varchar   NOT NULL,
    PRIMARY KEY (id)
);

INSERT INTO _Ref_S(name)
VALUES ('shipment1'),
       ('shipment1');
-- rollback drop table _Ref_S;

-- changeset zinov:2021-10-25-14-06-_Ref_L
CREATE TABLE IF NOT EXISTS _Ref_L
(
    id       bigserial NOT NULL,
    name     varchar   NOT NULL,
    order_id int       NOT NULL,
    foreign key (order_id) references _Ref_O (id) on delete cascade,
    PRIMARY KEY (id)
);

INSERT INTO _Ref_L(name, order_id)
VALUES ('lot1', 1),
       ('lot2', 2);
-- rollback drop table _Ref_L;

-- changeset zinov:2021-10-25-14-07-_Ref_M
CREATE TABLE IF NOT EXISTS _Ref_M
(
    id            bigserial NOT NULL,
    name          varchar   NOT NULL,
    type          varchar   NOT NULL,
    action        varchar   NOT NULL,
    event_trigger int,
    waiting_time  int,
    group_id      int       NOT NULL,
    PRIMARY KEY (id)
);

INSERT INTO _Ref_M(name, type, action, event_trigger, waiting_time, group_id)
VALUES ('node1', 'action', 'FirstInit', null, 0, 0),
       ('node2', 'action', 'SecondInit', null, 0, 0),
       ('node3', 'wait', 'Wait', null, 120, 500),
       ('node4', 'trigger', 'Trigger', 1, 0, 500),
       ('node5', 'terminate', 'Terminate', null, 0, 500);
-- rollback drop table _Ref_M;

-- changeset zinov:2021-10-25-14-11-1
CREATE TABLE _Ref_E
(
    id            bigserial primary key,
    name          varchar NOT NULL,
    event_type_id int,
    lot_id        int,
    foreign key (event_type_id) references _Ref_ET (id) on delete cascade,
    foreign key (lot_id) references _Ref_L (id) on delete cascade
);
INSERT INTO _Ref_E(name, event_type_id, lot_id)
VALUES ('event1', 1, 1),
       ('event2', 2, 2);
-- rollback drop table _Ref_E;

-- changeset zinov:2021-10-25-14-11-2
-- comment таблица процессинга (текущий шаг)
CREATE TABLE _InfoReg_CSR
(
    id            bigserial,
    lot_id        int REFERENCES _Ref_L (id) ON UPDATE CASCADE ON DELETE CASCADE,
    node_id       int REFERENCES _Ref_M (id) ON UPDATE CASCADE,
    thread        int NOT NULL             DEFAULT 1,
    weight        int not null             default 0,
    entry_time    timestamp WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    next_run_time timestamp WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT _InfoReg_CSR_pkey PRIMARY KEY (id, lot_id, node_id)
);
INSERT INTO _InfoReg_CSR(lot_id, node_id)
VALUES (1, 1),
       (2, 1);
-- rollback drop table _InfoReg_CSR;

-- changeset zinov:2021-10-25-14-11-3
-- comment табличная часть узла
CREATE TABLE _RefVT_ME
(
    id            bigserial,
    node_id       int REFERENCES _Ref_M (id) ON UPDATE CASCADE ON DELETE CASCADE,
    event_type_id int REFERENCES _Ref_ET (id) ON UPDATE CASCADE,
    CONSTRAINT _RefVT_ME_pkey PRIMARY KEY (node_id, event_type_id)
);
INSERT INTO _RefVT_ME(node_id, event_type_id)
VALUES (3, 1),
       (3, 2),
       (4, 1);
-- rollback drop table _RefVT_ME;

-- changeset zinov:2021-10-25-14-11-4
-- comment семафоры обработки событий, техн.
CREATE TABLE _InfoReg_ES
(
    id           bigserial,
    lot_id       int REFERENCES _Ref_L (id) ON UPDATE CASCADE ON DELETE CASCADE,
    semaphore_id int REFERENCES _Ref_ET (id) ON UPDATE CASCADE,
    event_id     int REFERENCES _Ref_E (id),
    entry_time   timestamp WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    order_id   int REFERENCES _Ref_O (id) ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT _InfoReg_ES_pkey PRIMARY KEY (id, lot_id, semaphore_id)
);
INSERT INTO _InfoReg_ES(lot_id, semaphore_id, event_id, order_id)
VALUES (1, 1, 1, 1),
       (2, 2, 2, 2);
-- rollback drop table _InfoReg_ES;

-- changeset zinov:2021-10-25-14-11-5
-- comment активность процессинга
CREATE TABLE _InfoReg_PA
(
    id         bigserial,
    order_id   int REFERENCES _Ref_O (id) ON UPDATE CASCADE ON DELETE CASCADE,
    thread_key varchar NOT NULL,
    start_time timestamp WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    thread_id  varchar NOT NULL,
    group_id   int     not null,
    CONSTRAINT _InfoReg_PA_pkey PRIMARY KEY (id, order_id, thread_key)
);
-- rollback drop table _InfoReg_PA;

-- changeset zinov:2021-10-25-14-11-6
-- comment группа обработки
CREATE TABLE _InfoReg_PG
(
    id       bigserial,
    order_id int REFERENCES _Ref_O (id) ON UPDATE CASCADE ON DELETE CASCADE,
    group_id int not null,
    CONSTRAINT _InfoReg_PG_pkey PRIMARY KEY (id, order_id)
);
INSERT INTO _InfoReg_PG(order_id, group_id) VALUES (1, 0), (2, 0);
-- rollback drop table _InfoReg_PG;