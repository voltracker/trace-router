CREATE TABLE hops (
  id UUID PRIMARY KEY,
  source_ip TEXT NOT NULL,
  dest_ip TEXT NOT NULL,
  latency DOUBLE PRECISION,
  time_added TIMESTAMP DEFAULT NOW()
);
