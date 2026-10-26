CREATE TABLE IF NOT EXISTS meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT ""
);

CREATE TABLE IF NOT EXISTS nodes (
    name         TEXT PRIMARY KEY,
    title        TEXT NOT NULL DEFAULT "",
    title_alt    TEXT NOT NULL DEFAULT "",
    parent       TEXT NOT NULL DEFAULT "",
    topics_count INTEGER NOT NULL DEFAULT 0,
    restricted   INTEGER NOT NULL DEFAULT 0,
    fetched_at   INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS topics (
    id             INTEGER PRIMARY KEY,
    node           TEXT NOT NULL DEFAULT "",
    node_title     TEXT NOT NULL DEFAULT "",
    title          TEXT NOT NULL DEFAULT "",
    author         TEXT NOT NULL DEFAULT "",
    created_at     INTEGER NOT NULL DEFAULT 0,
    last_touched_at INTEGER NOT NULL DEFAULT 0,
    replies        INTEGER NOT NULL DEFAULT -1,
    views          INTEGER NOT NULL DEFAULT 0,
    content_html   TEXT NOT NULL DEFAULT "",
    content_text   TEXT NOT NULL DEFAULT "",
    fetched_at     INTEGER NOT NULL DEFAULT 0,
    feed_at        INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_topics_created ON topics(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_topics_node_created ON topics(node, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_topics_last_touched ON topics(last_touched_at DESC);
CREATE INDEX IF NOT EXISTS idx_topics_author ON topics(author);
CREATE INDEX IF NOT EXISTS idx_topics_feed ON topics(feed_at);

CREATE TABLE IF NOT EXISTS replies (
    topic_id     INTEGER NOT NULL,
    floor        INTEGER NOT NULL,
    author       TEXT NOT NULL DEFAULT "",
    created_at   INTEGER NOT NULL DEFAULT 0,
    content_html TEXT NOT NULL DEFAULT "",
    content_text TEXT NOT NULL DEFAULT "",
    fetched_at   INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (topic_id, floor)
);
CREATE INDEX IF NOT EXISTS idx_replies_author ON replies(author);
CREATE INDEX IF NOT EXISTS idx_replies_created ON replies(topic_id, created_at);

CREATE TABLE IF NOT EXISTS list_pages (
    list_key   TEXT NOT NULL,
    page       INTEGER NOT NULL,
    entries    TEXT NOT NULL,
    pages      INTEGER NOT NULL DEFAULT 1,
    fetched_at INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (list_key, page)
);

CREATE TABLE IF NOT EXISTS feed_state (
    feed_key      TEXT PRIMARY KEY,
    etag          TEXT NOT NULL DEFAULT "",
    last_modified TEXT NOT NULL DEFAULT "",
    last_checked  INTEGER NOT NULL DEFAULT 0,
    last_success  INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS blocks (
    kind       TEXT NOT NULL,
    value      TEXT NOT NULL,
    created_at INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (kind, value)
);

CREATE TABLE IF NOT EXISTS summaries (
    topic_id   INTEGER PRIMARY KEY,
    model      TEXT NOT NULL DEFAULT "",
    content    TEXT NOT NULL DEFAULT "",
    created_at INTEGER NOT NULL DEFAULT 0
);
