CREATE TABLE IF NOT EXISTS notify_delivery_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    at TEXT NOT NULL,
    event TEXT NOT NULL,
    channel TEXT NOT NULL,
    state TEXT NOT NULL,
    detail TEXT NOT NULL
);
