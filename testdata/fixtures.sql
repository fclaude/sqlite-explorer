PRAGMA foreign_keys = ON;

CREATE TABLE customers (
    id    INTEGER PRIMARY KEY,
    name  TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE
);

CREATE TABLE orders (
    id          INTEGER PRIMARY KEY,
    customer_id INTEGER NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    total_cents INTEGER NOT NULL CHECK (total_cents >= 0),
    placed_at   TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE notes (
    id         INTEGER PRIMARY KEY,
    body       TEXT,
    payload    BLOB,
    updated_at TEXT
);

CREATE INDEX idx_orders_customer ON orders(customer_id);

CREATE VIEW orders_by_customer AS
SELECT c.id AS customer_id, c.name, COUNT(o.id) AS order_count, COALESCE(SUM(o.total_cents), 0) AS total_cents
FROM customers c
LEFT JOIN orders o ON o.customer_id = c.id
GROUP BY c.id, c.name;

CREATE TRIGGER notes_updated_at
AFTER UPDATE ON notes
FOR EACH ROW
BEGIN
    UPDATE notes SET updated_at = datetime('now') WHERE id = OLD.id;
END;

INSERT INTO customers (id, name, email) VALUES
    (1, 'Example Customer 001',   'customer001@example.invalid'),
    (2, 'Example Customer 002',    'customer002@example.invalid'),
    (3, 'Example Customer 003',   'customer003@example.invalid'),
    (4, 'Example Customer 004','customer004@example.invalid'),
    (5, 'Example Customer 005',   'customer005@example.invalid');

INSERT INTO orders (id, customer_id, total_cents) VALUES
    (1, 1, 1299),
    (2, 1,  499),
    (3, 2, 9999),
    (4, 3,    0),
    (5, 5, 12345);

INSERT INTO notes (id, body, payload) VALUES
    (1, 'plain text note', NULL),
    (2, NULL,              x'00010203deadbeef'),
    (3, 'note with "quotes", commas, and a newline'||x'0a'||'inside', NULL);
