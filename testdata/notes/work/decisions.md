# Decision log

## 2026-09-02: Billing database
Chose PostgreSQL 16 for the billing service. MongoDB was rejected because billing needs transactions across tables.

## 2026-09-20: API rate limit
The public API allows 100 requests per minute per key. Bursts above that get HTTP 429.

## 2026-10-05: Logging
All services log JSON lines. The log level in production is warn.
