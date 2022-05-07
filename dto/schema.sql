-- auto-generated definition
drop table if exists users;
create table users
(
    id         bigint unsigned auto_increment
        primary key comment '流水號',
    email      varchar(100) not null comment '電子郵件帳號',
    first_name varchar(45)  null comment '名',
    last_name  varchar(45)  null comment '姓',
    gender     tinyint      null comment '性別',
    created_at datetime(3)  null comment '建立時間',
    updated_at datetime(3)  null comment '修改時間',
    constraint idx_users_email
        unique (email)
);
