
create table global_roles (
    id smallserial primary key,
    name text not null
);

insert into global_roles (name) values
    ('guest'),
    ('student'),
    ('aitusa'),
    ('staff');

create table users (
    id uuid primary key,
    barcode text not null unique,
    username text not null unique,
    role_id smallint not null,
    first_name text not null,
    last_name text not null,
    avatar_source text not null,
    avatar_external text not null,
    avatar_s3_key text not null,
    email text not null unique,
    pass_hash bytea not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint users_role_id_fkey foreign key (role_id) references global_roles(id)
);

create table groups (
    id uuid primary key,
    name text not null,
    year text not null,
    major text not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create table students (
    user_id uuid primary key,
    group_id uuid not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint students_user_id_fkey foreign key (user_id) references users(id),
    constraint students_group_id_fkey foreign key (group_id) references groups(id)
);

create table staffs (
    user_id uuid primary key,
    constraint staffs_user_id_fkey foreign key (user_id) references users(id)
);

create table registrations (
    id uuid primary key,
    email text not null unique,
    status text not null,
    verification_code text not null,
    code_attempts smallint not null,
    code_expires_at timestamptz not null,
    resend_timeout timestamptz not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create table staff_invitations (
    id uuid primary key,
    creator_id uuid not null,
    code text not null unique,
    recipients_email text[],
    valid_from timestamptz,
    valid_until timestamptz,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    deleted_at timestamptz default null,
    constraint staff_invitations_creator_id_fkey foreign key (creator_id) references users(id)
);
