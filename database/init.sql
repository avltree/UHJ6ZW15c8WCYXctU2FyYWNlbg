create table if not exists objects
(
    id int auto_increment
        primary key,
    url text not null,
    `interval` int default 60 not null
);

create table if not exists response
(
    id int auto_increment
        primary key,
    created_at timestamp default CURRENT_TIMESTAMP not null,
    object_id int not null,
    duration decimal(4,3) not null,
    response text null,
    constraint response_objects_id_fk
        foreign key (object_id) references objects (id)
            on update cascade on delete cascade
);
