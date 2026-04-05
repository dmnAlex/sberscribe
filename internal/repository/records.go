package repository

import (
	"encoding/json"
	"iter"
	"time"

	"github.com/dmnAlex/sberscribe/internal/model"
	"github.com/dmnAlex/sberscribe/internal/storage/pg"
	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
)

const createRecordSQL = `
	INSERT INTO records (user_id, chat_id, bot_file_id, mime_type)
	VALUES (@userID, @chatID, @botFileID, @mimeType)
	RETURNING id
`

func (r *SberScribeRepo) CreateRecord(userID, chatID int64, botFileID, mimeType string) (int64, error) {
	args := pgx.NamedArgs{
		"userID":    userID,
		"chatID":    chatID,
		"botFileID": botFileID,
		"mimeType":  mimeType,
	}

	var id int64
	err := r.db.QueryRow(createRecordSQL, args, &id)
	return id, pg.WrapAlreadyExists(err)
}

const getRecordSQL = `
	SELECT id, user_id, chat_id, bot_file_id, mime_type, upload_file_id, task_id, download_file_id, content, title, summary, raw, status, attempts
	FROM records
	WHERE user_id = @userID AND id = @ID
`

func (r *SberScribeRepo) GetRecord(userID, id int64) (model.Record, error) {
	args := pgx.NamedArgs{
		"userID": userID,
		"ID":     id,
	}

	var record model.Record
	err := r.db.QueryRow(getRecordSQL, args, record.AsIfaceList()...)
	return record, pg.WrapNotFound(err)
}

const getRecordsByUserIDSQL = `
	SELECT id, user_id, chat_id, bot_file_id, mime_type, upload_file_id, task_id, download_file_id, content, title, summary, raw, status, attempts
	FROM records
	WHERE user_id = @userID AND status = @status
	ORDER BY created_at
`

func (r *SberScribeRepo) GetRecordsByUserID(userID int64) iter.Seq2[model.Record, error] {
	args := pgx.NamedArgs{
		"userID": userID,
		"status": model.StatusSummarized,
	}

	return pg.QueryMany(r.db, getRecordsByUserIDSQL, pg.IfaceListFunc[*model.Record](), args)
}

const findRecordsSQL = `
	SELECT id, user_id, chat_id, bot_file_id, mime_type, upload_file_id, task_id, download_file_id, content, title, summary, raw, status, attempts
	FROM records
	WHERE user_id = @userID AND status = @status AND content_tsv @@ websearch_to_tsquery('russian', @query)
	ORDER BY ts_rank(content_tsv, websearch_to_tsquery('russian', @query)) DESC, created_at DESC
`

func (r *SberScribeRepo) FindRecords(userID int64, query string) iter.Seq2[model.Record, error] {
	args := pgx.NamedArgs{
		"userID": userID,
		"status": model.StatusSummarized,
		"query":  query,
	}

	return pg.QueryMany(r.db, findRecordsSQL, pg.IfaceListFunc[*model.Record](), args)
}

const updateRecordUploadFileIDSQL = `
	UPDATE records
	SET upload_file_id = @uploadFileID,
		status = status + 1,
		attempts = 0,
		updated_at = NOW()
	WHERE id = @ID
`

func (r *SberScribeRepo) UpdateRecordUploadFileID(ID int64, uploadFileID string) error {
	args := pgx.NamedArgs{
		"ID":           ID,
		"uploadFileID": uploadFileID,
	}

	res, err := r.db.Exec(updateRecordUploadFileIDSQL, args)
	return pg.HandleExecResult(res, err)
}

const updateRecordTaskIDSQL = `
	UPDATE records
	SET task_id = @taskID,
		status = status + 1,
		attempts = 0,
		updated_at = NOW()
	WHERE id = @ID
`

func (r *SberScribeRepo) UpdateRecordTaskID(ID int64, taskID string) error {
	args := pgx.NamedArgs{
		"ID":     ID,
		"taskID": taskID,
	}

	res, err := r.db.Exec(updateRecordTaskIDSQL, args)
	return pg.HandleExecResult(res, err)
}

const updateRecordDownloadFileIDSQL = `
	UPDATE records
	SET download_file_id = @downloadFileID,
		status = status + 1,
		attempts = 0,
		updated_at = NOW()
	WHERE id = @ID
`

func (r *SberScribeRepo) UpdateRecordDownloadFileID(ID int64, downloadFileID string) error {
	args := pgx.NamedArgs{
		"ID":             ID,
		"downloadFileID": downloadFileID,
	}

	res, err := r.db.Exec(updateRecordDownloadFileIDSQL, args)
	return pg.HandleExecResult(res, err)
}

const updateRecordContentAndRawSQL = `
	UPDATE records
	SET content = @content,
		raw = @raw,
		status = status + 1,
		attempts = 0,
		updated_at = NOW()
	WHERE id = @ID
`

func (r *SberScribeRepo) UpdateRecordContentAndRaw(ID int64, content string, raw json.RawMessage) error {
	args := pgx.NamedArgs{
		"ID":      ID,
		"content": content,
		"raw":     raw,
	}

	res, err := r.db.Exec(updateRecordContentAndRawSQL, args)
	return pg.HandleExecResult(res, err)
}

const updateRecordTitleAndSummarySQL = `
	UPDATE records
	SET title = @title,
		summary = @summary,
		status = status + 1,
		attempts = 0,
		updated_at = NOW()
	WHERE id = @ID
`

func (r *SberScribeRepo) UpdateRecordTitleAndSummary(ID int64, title, summary string) error {
	args := pgx.NamedArgs{
		"ID":      ID,
		"title":   title,
		"summary": summary,
	}

	res, err := r.db.Exec(updateRecordTitleAndSummarySQL, args)
	return pg.HandleExecResult(res, err)
}

const rollbackRecordStatusSQL = `
	UPDATE records
	SET status = status - 1,
		updated_at = NOW()
	WHERE id = @ID
`

func (r *SberScribeRepo) RollbackRecordStatus(ID int64) error {
	args := pgx.NamedArgs{
		"ID": ID,
	}

	res, err := r.db.Exec(rollbackRecordStatusSQL, args)
	return pg.HandleExecResult(res, err)
}

const getNextRecordsBatchSQL = `
	UPDATE records r
	SET status = status + 1,
			attempts = attempts + 1,
			updated_at = NOW()
	FROM (
			SELECT id
			FROM records
			WHERE status != @finalStatus
				AND status % 2 = 0
			ORDER BY attempts ASC, updated_at ASC
			LIMIT @batchSize
			FOR UPDATE SKIP LOCKED
	) AS sub
	WHERE r.id = sub.id
	RETURNING r.id, r.user_id, r.chat_id, r.bot_file_id, r.mime_type, r.upload_file_id, r.task_id,
		r.download_file_id, r.content, r.title, r.summary, r.raw, r.status, r.attempts
`

func (r *SberScribeRepo) GetNextRecordsBatch(batchSize int) iter.Seq2[model.Record, error] {
	args := pgx.NamedArgs{
		"batchSize":   batchSize,
		"finalStatus": model.StatusSummarized,
	}

	return pg.QueryMany(r.db, getNextRecordsBatchSQL, pg.IfaceListFunc[*model.Record](), args)
}

const releaseStaleRecordsSQL = `
	UPDATE records
	SET status = status - 1,
		updated_at = NOW()
	WHERE status % 2 = 1
		AND updated_at <= @staleThreshold
`

func (r *SberScribeRepo) ReleaseStaleRecords(staleThreshold time.Time) error {
	args := pgx.NamedArgs{
		"staleThreshold": staleThreshold,
	}

	_, err := r.db.Exec(releaseStaleRecordsSQL, args)
	return errors.Wrap(err, "exec")
}

const deleteRecordSQL = `
	DELETE FROM records WHERE id = @ID
`

func (r *SberScribeRepo) DeleteRecord(ID int64) error {
	args := pgx.NamedArgs{
		"ID": ID,
	}

	res, err := r.db.Exec(deleteRecordSQL, args)
	return pg.HandleExecResult(res, err)
}
