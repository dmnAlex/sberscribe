package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dmnAlex/sberscribe/internal/logger"
	"github.com/dmnAlex/sberscribe/internal/model"
	"github.com/dmnAlex/sberscribe/internal/model/errx"
	"github.com/pkg/errors"
)

func (s *SberScribeService) recordBatchWorker(ctx context.Context) error {
	logger.Log.Debug("start record batch worker")
	for {
		batch, err := s.repo.GetNextRecordsBatch(batchSize)
		if err != nil && !errors.Is(err, errx.ErrNotFound) {
			logger.Log.Error("get next record batch", "error", err)
		}

		for _, record := range batch {
			s.recordCh <- record
		}

		select {
		case <-ctx.Done():
			logger.Log.Debug("stop record batch worker")
			return nil
		case <-time.After(pollDelay):
		}
	}
}

func (s *SberScribeService) recordWorker(ctx context.Context, ID int) error {
	logger.Log.Debug("start record worker", "id", ID)
	for {
		select {
		case <-ctx.Done():
			logger.Log.Debug("stop record worker", "id", ID)
			return nil
		case record := <-s.recordCh:
			if err := s.processRecord(record); err != nil {
				logger.Log.Error("process record", "error", err)
			}
		}
	}
}

func (s *SberScribeService) staleRecordWorker(ctx context.Context) error {
	logger.Log.Debug("start stale record worker")
	for {
		threshold := time.Now().Add(-staleTime)
		if err := s.repo.ReleaseStaleRecords(threshold); err != nil {
			logger.Log.Error("release stale records", "error", err)
		}

		select {
		case <-ctx.Done():
			logger.Log.Debug("stop stale record worker")
			return nil
		case <-time.After(pollDelay):
		}
	}
}

func (s *SberScribeService) processRecord(record model.Record) error {
	if record.Status == model.StatusPolling && record.Attempts > pollAttemptsLimit || record.Attempts > defaultAttemptsLimit {
		if err := s.repo.DeleteRecord(record.ID); err != nil {
			return errors.Wrap(err, "delete record")
		}

		msg := fmt.Sprintf("Превышено максимальное количество попыток распознат запись %d. Попробуйте загрузить ее заново.", record.ID)
		s.bot.OutCh() <- model.OutTask{ChatID: record.ChatID, Message: msg}
		return nil
	}

	var err error
	switch record.Status {
	case model.StatusUploading:
		err = s.recordUploading(record)
	case model.StatusRecognizing:
		err = s.recordRecognizing(record)
	case model.StatusPolling:
		err = s.recordPolling(record)
	case model.StatusDownloading:
		err = s.recordDownloading(record)
	case model.StatusSummarizing:
		err = s.recordSummarize(record)
	default:
		return errors.Errorf("incorrect record status: %d", record.Status)
	}

	if err != nil {
		if err := s.repo.RollbackRecordStatus(record.ID); err != nil {
			return errors.Wrap(err, "rollback record status")
		}
		logger.Log.Error("process record", "error", err, "id", record.ID, "status", record.Status, "attempts", record.Attempts)
		return nil
	}

	if record.Status == model.StatusSummarizing {
		msg := fmt.Sprintf("Распознавание записи %d завершено успешно.", record.ID)
		s.bot.OutCh() <- model.OutTask{ChatID: record.ChatID, Message: msg}
	}

	return nil
}

func (s *SberScribeService) recordUploading(record model.Record) error {
	data, err := s.bot.GetFileRC(record.BotFileID)
	if err != nil {
		return errors.Wrap(err, "get file reader")
	}
	defer data.Close()

	uploadFileID, err := s.salute.Upload(s.stopCtx, data)
	if err != nil {
		return errors.Wrap(err, "salute upload")
	}

	if err := s.repo.UpdateRecordUploadFileID(record.ID, uploadFileID); err != nil {
		return errors.Wrap(err, "upldate record upload file id")
	}

	return nil
}

func (s *SberScribeService) recordRecognizing(record model.Record) error {
	if record.UploadFileID == nil {
		return errors.New("upload file id is nil")
	}

	taskID, err := s.salute.Recognize(s.stopCtx, *record.UploadFileID, record.MimeType)
	if err != nil {
		return errors.Wrap(err, "salute recognize")
	}

	if err := s.repo.UpdateRecordTaskID(record.ID, taskID); err != nil {
		return errors.Wrap(err, "update record task id")
	}

	return nil
}

func (s *SberScribeService) recordPolling(record model.Record) error {
	if record.TaskID == nil {
		return errors.New("task id is nil")
	}

	downloadFileID, err := s.salute.PollTask(s.stopCtx, *record.TaskID)
	if err != nil {
		return errors.Wrap(err, "salute poll task")
	}

	if err := s.repo.UpdateRecordDownloadFileID(record.ID, downloadFileID); err != nil {
		return errors.Wrap(err, "update record download file id")
	}

	return nil
}

func (s *SberScribeService) recordDownloading(record model.Record) error {
	if record.DownloadFileID == nil {
		return errors.New("download file id is nil")
	}

	content, raw, err := s.salute.Download(s.stopCtx, *record.DownloadFileID)
	if err != nil {
		return errors.Wrap(err, "salute download")
	}

	if err := s.repo.UpdateRecordContentAndRaw(record.ID, content, raw); err != nil {
		return errors.Wrap(err, "salute update record content and raw")
	}

	return nil
}

const summarizePrompt = `
	Следующим сообщением от пользователя ты получишь транскрипцию аудио.
	Твоя задача сформулировать название для содержимого и краткую выжимку из содержимого.
	В твоем ответе должен быть только json с двумя полями:
	- Поле title содержащее название
	- Поле summary содержащее краткую выжимку
`

func (s *SberScribeService) recordSummarize(record model.Record) error {
	if record.Content == nil {
		return errors.New("content is nil")
	}
	if record.Raw == nil {
		return errors.New("raw is nil")
	}

	msgs := []model.ChatMessage{{
		Role:    model.SystemRole,
		Content: summarizePrompt,
	}}

	msgs = append(msgs, model.ChatMessage{
		Role:    model.UserRole,
		Content: *record.Content,
	})

	answer, err := s.giga.Chat(s.stopCtx, msgs)
	if err != nil {
		return errors.Wrap(err, "chat")
	}

	var res model.SummarizeResult
	if err := json.Unmarshal([]byte(answer), &res); err != nil {
		return errors.Wrap(err, "unmarshal json")
	}

	if err := s.repo.UpdateRecordTitleAndSummary(record.ID, res.Title, res.Summary); err != nil {
		return errors.Wrap(err, "update record title and summary")
	}

	return nil
}
