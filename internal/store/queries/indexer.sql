-- Indexer work queue. Rule: NO transaction or row lock is held while calling the
-- gateway. A claim is one short statement that commits immediately and sets a
-- lease; a crashed worker's lease simply expires (INDEXER_LEASE_TIMEOUT).
-- Every write by a worker is fenced on claimed_by, so a worker whose lease was
-- taken over cannot overwrite the new owner's result.

-- name: FailExhaustedVersions :many
-- Run before each claim. Versions that used up INDEXER_MAX_ATTEMPTS (including
-- ones whose worker died on the last attempt) become terminal 'failed' -> alert.
UPDATE document_versions
SET index_status = 'failed',
    index_error  = COALESCE(index_error, 'max attempts exceeded'),
    claimed_at   = NULL,
    claimed_by   = NULL
WHERE index_attempts >= sqlc.arg(max_attempts)::int
  AND (index_status = 'pending'
       OR (index_status = 'indexing'
           AND claimed_at < now() - make_interval(secs => sqlc.arg(lease_seconds)::int)))
RETURNING id, document_id, index_attempts, index_error;

-- name: ClaimVersionForIndexing :one
-- Claims the oldest runnable version: pending and due, or indexing with an
-- expired lease. index_attempts is incremented at claim time, so a version that
-- keeps killing its worker still reaches max_attempts.
UPDATE document_versions
SET index_status = 'indexing',
    claimed_at   = now(),
    claimed_by   = sqlc.arg(worker_id)::text,
    index_attempts = index_attempts + 1
WHERE id = (
    SELECT v.id
    FROM document_versions v
    WHERE v.parse_status <> 'failed'
      AND v.index_attempts < sqlc.arg(max_attempts)::int
      AND ((v.index_status = 'pending' AND v.index_next_attempt_at <= now())
           OR (v.index_status = 'indexing'
               AND v.claimed_at < now() - make_interval(secs => sqlc.arg(lease_seconds)::int)))
    ORDER BY v.index_next_attempt_at, v.id
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING *;

-- name: ExtendLease :execrows
-- Heartbeat for long versions; 0 rows = the lease was lost, abandon the work.
UPDATE document_versions
SET claimed_at = now()
WHERE id = $1 AND index_status = 'indexing' AND claimed_by = $2;

-- name: CompleteIndexing :execrows
-- Fenced: 0 rows = the lease was lost; discard this worker's result.
UPDATE document_versions
SET index_status = 'done', index_error = NULL, claimed_at = NULL, claimed_by = NULL
WHERE id = $1 AND index_status = 'indexing' AND claimed_by = $2;

-- name: ReleaseForRetry :execrows
-- Transient failure: back to the queue after a backoff computed by the worker
-- (e.g. base * 2^attempts, capped).
UPDATE document_versions
SET index_status = 'pending',
    index_error = sqlc.arg(error)::text,
    claimed_at = NULL,
    claimed_by = NULL,
    index_next_attempt_at = now() + make_interval(secs => sqlc.arg(backoff_seconds)::int)
WHERE id = sqlc.arg(id) AND index_status = 'indexing' AND claimed_by = sqlc.arg(worker_id)::text;

-- name: FailIndexing :execrows
-- Permanent failure (e.g. unparseable content): terminal, raises the alert.
UPDATE document_versions
SET index_status = 'failed', index_error = sqlc.arg(error)::text, claimed_at = NULL, claimed_by = NULL
WHERE id = sqlc.arg(id) AND index_status = 'indexing' AND claimed_by = sqlc.arg(worker_id)::text;

-- name: CountIndexerBacklog :one
-- Feeds the index_pending_versions / oldest-pending-age gauges and the alert.
SELECT count(*) FILTER (WHERE index_status IN ('pending', 'indexing')) AS runnable,
       count(*) FILTER (WHERE index_status = 'failed')                 AS failed,
       COALESCE(extract(epoch FROM now() - min(fetched_at) FILTER (WHERE index_status IN ('pending', 'indexing'))), 0)::bigint AS oldest_pending_seconds
FROM document_versions;
