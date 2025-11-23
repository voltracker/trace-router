CREATE TABLE geoip (
  id UUID PRIMARY KEY,
  ip_addr INET NOT NULL UNIQUE,
  lat DOUBLE PRECISION,
  lon DOUBLE PRECISION,
  isp TEXT,
  org TEXT,
  asn TEXT
);

CREATE TABLE hops (
  id UUID PRIMARY KEY,
  source_ip INET NOT NULL REFERENCES geoip(ip_addr),
  dest_ip INET NOT NULL REFERENCES geoip(ip_addr),
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
