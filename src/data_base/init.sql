-- init.sql

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS employee (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    first_name VARCHAR(50),
    last_name VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TYPE organization_type AS ENUM (
    'IE',
    'LLC',
    'JSC'
);

CREATE TABLE IF NOT EXISTS organization (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    type organization_type,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS organization_responsible (
    id SERIAL PRIMARY KEY,
    organization_id INT REFERENCES organization(id) ON DELETE CASCADE,
    user_id INT REFERENCES employee(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS tenders (
    id UUID PRIMARY KEY DEFAULT (uuid_generate_v4()),
    name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    status VARCHAR(20) NOT NULL,
    service_type VARCHAR(50) NOT NULL,
    author_id INT NOT NULL,
    organization_id INT NOT NULL,      
    version INT DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS tenders_archive (
    unique_id SERIAL PRIMARY KEY,
    id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    status VARCHAR(20) NOT NULL,
    service_type VARCHAR(50) NOT NULL,
    version INT NOT NULL
);

CREATE TABLE IF NOT EXISTS bids (
    id UUID PRIMARY KEY DEFAULT (uuid_generate_v4()),
    name VARCHAR(100) NOT NULL,
    description VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL,
    author_type VARCHAR(20) NOT NULL,
    author_id INT NOT NULL,
    tender_id UUID NOT NULL,
    version INT DEFAULT 1,
    approve_count INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS bids_reviews (
    id SERIAL PRIMARY KEY,
    bid_id UUID NOT NULL,
    author_name VARCHAR(50) NOT NULL,
    description VARCHAR(1000) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);


CREATE TABLE IF NOT EXISTS bids_archive (
    unique_id SERIAL PRIMARY KEY,
    id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(100) NOT NULL,
    version INT NOT NULL
);

-- For testing

INSERT INTO employee (username, first_name, last_name) VALUES
    ('user1', 'John', 'Doe'),
    ('user2', 'Jane', 'Smith'),
    ('user3', 'Alice', 'Johnson'),
    ('user4', 'Bob', 'Brown'),
    ('user5', 'Charlie', 'Davis'),
    ('user6', 'Charlie', 'Davis');


INSERT INTO organization (name, description, type) VALUES
    ('Organization A', 'This is organization A', 'LLC'),
    ('Organization B', 'This is organization B', 'IE'),
    ('Organization C', 'This is organization C', 'JSC');


INSERT INTO organization_responsible (organization_id, user_id) VALUES
    (1, 1),
    (1, 2),
    (2, 3),
    (3, 4),
    (3, 5),
    (1, 6);


INSERT INTO tenders (name, description, status, service_type, author_id, organization_id, created_at)
VALUES 
    ('tender A', 'Описание тендера A', 'Created', 'Delivery', 1, 1, NOW()),
    ('tender B', 'Описание тендера B', 'In Progress', 'Delivery', 2, 2, NOW()),
    ('tender C', 'Описание тендера C', 'Completed', 'Delivery', 3, 3, NOW()),
    ('tender D', 'Описание тендера D', 'Canceled', 'Delivery', 4, 3, NOW()),
    ('tender E', 'Описание тендера E', 'Created', 'Delivery', 5, 3, NOW());

INSERT INTO bids (name, description, status, author_type, author_id, tender_id, version, created_at)
VALUES 
    ('Доставка товаров Алексей', 'Описание', 'Created', 'User', 1, (SELECT id FROM tenders WHERE name = 'tender A'), 1, NOW()),
    ('Предложение по стройматериалам', 'Описание', 'Published', 'Organization', 1, (SELECT id FROM tenders WHERE name = 'tender B'), 1, NOW()),
    ('Услуги по уборке', 'Описание', 'Created', 'User', 2, (SELECT id FROM tenders WHERE name = 'tender C'), 1, NOW()),
    ('Проектирование зданий', 'Описание', 'Canceled', 'Organization', 3, (SELECT id FROM tenders WHERE name = 'tender D'), 1, NOW()),
    ('Консультационные услуги', 'Описание', 'Created', 'User', 4, (SELECT id FROM tenders WHERE name = 'tender E'), 1, NOW());