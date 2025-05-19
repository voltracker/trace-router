CREATE TABLE hops (
  id UUID PRIMARY KEY,
  source_ip TEXT NOT NULL,
  dest_ip TEXT NOT NULL,
  latency DOUBLE PRECISION,
  time_added TIMESTAMP DEFAULT NOW()
);

CREATE VIEW hops_agg AS
SELECT
  source_ip,
  dest_ip,
  COUNT(*)       AS count,
  AVG(latency)   AS avg_latency
FROM hops 
GROUP BY source_ip, dest_ip;
