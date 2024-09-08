-- init.sql

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

    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    status VARCHAR(20) NOT NULL,
    service_type VARCHAR(50) NOT NULL,
    author_id UUID NOT NULL,      
    author_type VARCHAR(20) NOT NULL,
    version INT DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP

);


CREATE TABLE IF NOT EXISTS bids (

    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL,
    author_type VARCHAR(20) NOT NULL,
    author_id UUID NOT NULL,
    tender_id UUID NOT NULL REFERENCES tenders(id) ON DELETE CASCADE,
    version INT DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- For testing

INSERT INTO employee (username, first_name, last_name) VALUES
    ('user1', 'John', 'Doe'),
    ('user2', 'Jane', 'Smith'),
    ('user3', 'Alice', 'Johnson'),
    ('user4', 'Bob', 'Brown'),
    ('user5', 'Charlie', 'Davis');



INSERT INTO organization (name, description, type) VALUES
    ('Organization A', 'This is organization A', 'LLC'),
    ('Organization B', 'This is organization B', 'IE'),
    ('Organization C', 'This is organization C', 'JSC');


INSERT INTO organization_responsible (organization_id, user_id) VALUES
    (1, 1),
    (1, 2),
    (2, 3),
    (3, 4),
    (3, 5);

INSERT INTO tenders (id, name, description, status,service_type, author_id, author_type, created_at)
VALUES 
    (gen_random_uuid(), 'tender A', 'Описание тендера A', 'Created','Delivery', gen_random_uuid(), 'User', NOW()),
    (gen_random_uuid(), 'tender B', 'Описание тендера B', 'In Progress','Delivery', gen_random_uuid(), 'Organization', NOW()),
    (gen_random_uuid(), 'tender C', 'Описание тендера C', 'Completed','Delivery', gen_random_uuid(), 'User', NOW()),
    (gen_random_uuid(), 'tender D', 'Описание тендера D', 'Canceled','Delivery', gen_random_uuid(), 'Organization', NOW()),
    (gen_random_uuid(), 'tender E', 'Описание тендера E', 'Created', 'Delivery',gen_random_uuid(), 'User', NOW());


INSERT INTO bids (name, status, author_type, author_id, tender_id, version, created_at)
VALUES 
    ('Доставка товаров Алексей', 'Created', 'User', gen_random_uuid(), (SELECT id FROM tenders WHERE name = 'tender A'), 1, NOW()),
    ('Предложение по стройматериалам', 'Published', 'Organization', gen_random_uuid(), (SELECT id FROM tenders WHERE name = 'tender B'), 1, NOW()),
    ('Услуги по уборке', 'Created', 'User', gen_random_uuid(), (SELECT id FROM tenders WHERE name = 'tender C'), 1, NOW()),
    ('Проектирование зданий', 'Canceled', 'Organization', gen_random_uuid(), (SELECT id FROM tenders WHERE name = 'tender D'), 1, NOW()),
    ('Консультационные услуги', 'Created', 'User', gen_random_uuid(), (SELECT id FROM tenders WHERE name = 'tender E'), 1, NOW());