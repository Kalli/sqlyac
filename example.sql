---
-- @name CreateUsersTable
CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100) UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    active BOOLEAN DEFAULT TRUE
);
---

---
-- @name CreateOrdersTable
CREATE TABLE orders (
    id INTEGER PRIMARY KEY,
    user_id INTEGER,
    total_amount DECIMAL(10,2),
    status VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

---
-- @name InsertSampleUsers
INSERT INTO users (username, email) VALUES 
    ('alice', 'alice@example.com'),
    ('bob', 'bob@example.com'),
    ('charlie', 'charlie@example.com'),
    ('diana', 'diana@example.com');

---
-- @name InsertSampleOrders
INSERT INTO orders (user_id, total_amount, status) VALUES 
    (1, 29.99, 'completed'),
    (1, 15.50, 'completed'),
    (2, 450.00, 'pending'),
    (2, 89.99, 'completed'),
    (3, 1200.00, 'completed'),
    (4, 25.00, 'cancelled');

---
-- @name GetAllUsers
SELECT * FROM users ORDER BY created_at;

---
-- @name GetActiveUsers
SELECT id, username, email 
FROM users 
WHERE active = TRUE 
ORDER BY username;

---
-- @name GetLargeOrders
SELECT o.id, u.username, o.total_amount, o.status
FROM orders o
JOIN users u ON o.user_id = u.id
WHERE o.total_amount > 100
ORDER BY o.total_amount DESC;

---
-- @name GetUserOrderSummary
SELECT 
    u.username,
    COUNT(o.id) as order_count,
    COALESCE(SUM(o.total_amount), 0) as total_spent,
    COALESCE(AVG(o.total_amount), 0) as avg_order_value
FROM users u
LEFT JOIN orders o ON u.id = o.user_id
GROUP BY u.id, u.username
ORDER BY total_spent DESC;

---
-- @name GetRecentOrders
SELECT 
    o.id,
    u.username,
    o.total_amount,
    o.status,
    o.created_at
FROM orders o
JOIN users u ON o.user_id = u.id
WHERE o.created_at > DATE('now', '-7 days')
ORDER BY o.created_at DESC;

---
-- @name CountOrdersByStatus
SELECT status, COUNT(*) as count
FROM orders 
GROUP BY status
ORDER BY count DESC;

---
-- @name CleanupTestData
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS users;
