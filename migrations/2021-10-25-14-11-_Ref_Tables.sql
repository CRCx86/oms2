-- liquibase formatted sql

-- changeset zinov:2021-10-25-14-11-1
CREATE TABLE _Ref_E
(
    id bigserial primary key,
    name varchar NOT NULL,
    event_type_id int,
    lot_id int,
    foreign key(event_type_id) references _Ref_ET(id) on delete cascade,
    foreign key(lot_id) references _Ref_L(id) on delete cascade
);
INSERT INTO _Ref_E(name, event_type_id, lot_id)
VALUES('event1', 1, 1), ('event2', 2, 2);
-- rollback drop table _Ref_E;

-- changeset zinov:2021-10-25-14-11-2
-- comment таблица процессинга (текущий шаг)
CREATE TABLE _InfoReg_CSR
(
    id bigserial,
    lot_id    int REFERENCES _Ref_L (id) ON UPDATE CASCADE ON DELETE CASCADE,
    node_id 	int REFERENCES _Ref_M (id) ON UPDATE CASCADE,
    thread	int NOT NULL DEFAULT 1,
    entry_time timestamp WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT _InfoReg_CSR_pkey PRIMARY KEY (lot_id, node_id)
);
INSERT INTO _InfoReg_CSR(lot_id, node_id)
VALUES(1, 1), (2, 1);
-- rollback drop table _InfoReg_CSR;

-- changeset zinov:2021-10-25-14-11-3
-- comment табличная часть узла
CREATE TABLE _RefVT_ME
(
    id 			bigserial,
    node_id    	int REFERENCES _Ref_M (id) ON UPDATE CASCADE ON DELETE CASCADE,
    event_type_id int REFERENCES _Ref_ET (id) ON UPDATE CASCADE,
    CONSTRAINT _RefVT_ME_pkey PRIMARY KEY (node_id, event_type_id)
);
INSERT INTO _RefVT_ME(node_id, event_type_id)
VALUES(3, 1), (3, 2), (4, 1);
-- rollback drop table _RefVT_ME;

-- changeset zinov:2021-10-25-14-11-4
-- comment семафоры обработки событий, техн.
CREATE TABLE _InfoReg_ES
(
    id bigserial,
    lot_id    int REFERENCES _Ref_L (id) ON UPDATE CASCADE ON DELETE CASCADE,
    semaphore_id int REFERENCES _Ref_ET (id) ON UPDATE CASCADE,
    event_id  int REFERENCES _Ref_E (id),
    CONSTRAINT _InfoReg_ES_pkey PRIMARY KEY (lot_id, semaphore_id)
);
INSERT INTO _InfoReg_ES(lot_id, semaphore_id, event_id)
VALUES(1, 1, 1), (2, 2, 2);
-- rollback drop table _InfoReg_ES;