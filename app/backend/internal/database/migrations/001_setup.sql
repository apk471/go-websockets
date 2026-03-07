CREATE TYPE match_status AS ENUM ('scheduled', 'live', 'finished');

CREATE TABLE matches (
    id SERIAL PRIMARY KEY,
    sport TEXT NOT NULL,
    home_team TEXT NOT NULL,
    away_team TEXT NOT NULL,
    status match_status NOT NULL DEFAULT 'scheduled',
    start_time TIMESTAMP,
    end_time TIMESTAMP,
    home_score INTEGER NOT NULL DEFAULT 0,
    away_score INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE commentary (
    id SERIAL PRIMARY KEY,
    match_id INTEGER NOT NULL REFERENCES matches (id),
    minute INTEGER,
    sequence INTEGER,
    period TEXT,
    event_type TEXT,
    actor TEXT,
    team TEXT,
    message TEXT NOT NULL,
    metadata JSONB,
    tags TEXT[],
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_commentary_match_id ON commentary (match_id);
CREATE INDEX idx_commentary_match_sequence ON commentary (match_id, sequence);

---- create above / drop below ----

DROP INDEX IF EXISTS idx_commentary_match_sequence;
DROP INDEX IF EXISTS idx_commentary_match_id;
DROP TABLE IF EXISTS commentary;
DROP TABLE IF EXISTS matches;
DROP TYPE IF EXISTS match_status;
