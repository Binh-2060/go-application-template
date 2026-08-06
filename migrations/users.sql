create table users
(
    id         uuid                     default gen_random_uuid()        not null
        constraint users_pk
            primary key,
    name       varchar(200)             default 'N/A'::character varying not null,
    created_at timestamp with time zone default now()                    not null,
    surename   varchar(200)                                              not null
);

alter table users
    owner to postgres;

