package service

import (
	"context"
	"time"

	"github.com/dmnAlex/sberscribe/internal/bot"
	"github.com/dmnAlex/sberscribe/internal/gigachat"
	"github.com/dmnAlex/sberscribe/internal/model"
	"github.com/dmnAlex/sberscribe/internal/repository"
	"github.com/dmnAlex/sberscribe/internal/salutespeech"
	"golang.org/x/sync/errgroup"
)

const (
	chanSize             = 100
	batchSize            = 50
	pollDelay            = 3 * time.Second
	defaultAttemptsLimit = 10
	pollAttemptsLimit    = 50
	recordWorkersCount   = 10
	taskWorkerCount      = 10
	staleTime            = 3 * time.Minute
)

type SberScribeService struct {
	stopCtx  context.Context
	eg       *errgroup.Group
	repo     repository.Repository
	salute   *salutespeech.Client
	giga     *gigachat.Client
	bot      *bot.Bot
	recordCh chan model.Record
}

func New(ctx context.Context, repo repository.Repository, salute *salutespeech.Client, giga *gigachat.Client, bot *bot.Bot) (*SberScribeService, error) {
	return &SberScribeService{
		stopCtx:  ctx,
		repo:     repo,
		salute:   salute,
		giga:     giga,
		bot:      bot,
		recordCh: make(chan model.Record, chanSize),
	}, nil
}

func (s *SberScribeService) StartWorkers() {
	var ctx context.Context
	s.eg, ctx = errgroup.WithContext(s.stopCtx)

	s.eg.Go(func() error {
		return s.recordBatchWorker(ctx)
	})

	s.eg.Go(func() error {
		return s.staleRecordWorker(ctx)
	})

	for i := range recordWorkersCount {
		s.eg.Go(func() error {
			return s.recordWorker(ctx, i)
		})
	}

	for i := range taskWorkerCount {
		s.eg.Go(func() error {
			return s.taskWorker(ctx, i)
		})
	}
}

func (s *SberScribeService) Stop() {
	s.bot.Stop()
	s.eg.Wait()
	s.salute.Close()
	s.giga.Close()
	s.repo.Close()
}
