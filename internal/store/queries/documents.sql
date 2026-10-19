-- name: GetDocumentByURL :one
SELECT * FROM documents WHERE canonical_url = $1;

-- name: ListServableDeadlines :many
-- The only read path for deadline answers. servable_deadlines (migration 00008)
-- is the single definition of "may be shown": is_current AND quote_verified AND
-- date_in_quote AND review_status <> 'rejected'. Never query `deadlines` directly
-- for serving.
SELECT * FROM servable_deadlines
WHERE academic_year = $1
ORDER BY ends_on NULLS LAST, event_key;
