CREATE TABLE IF NOT EXISTS proxy_publication_commit (
    id INTEGER PRIMARY KEY CHECK(id = 1),
    token TEXT NOT NULL
);
