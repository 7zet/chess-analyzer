package jobs

import (
	"context"
	"errors"
	"log"

	chessparser "github.com/ahror/royal-move/internal/chess"
	"github.com/ahror/royal-move/internal/config"
	"github.com/ahror/royal-move/internal/engine"
	"github.com/ahror/royal-move/internal/store"
)

type Runner struct {
	cfg   config.Config
	store *store.Store
	queue chan string
}

var ErrQueueFull = errors.New("analysis queue is full")

func New(cfg config.Config, store *store.Store) *Runner {
	return &Runner{cfg: cfg, store: store, queue: make(chan string, cfg.QueueCapacity)}
}
func (r *Runner) Enqueue(id string) error {
	select {
	case r.queue <- id:
		return nil
	default:
		return ErrQueueFull
	}
}
func (r *Runner) Start(ctx context.Context) {
	ids, err := r.store.QueuedAnalysisIDs(ctx)
	if err != nil {
		log.Printf("recover analysis queue: %v", err)
	}
	for _, id := range ids {
		if err = r.Enqueue(id); err != nil {
			log.Printf("recover job %s: %v", id, err)
			break
		}
	}
	for worker := 0; worker < r.cfg.AnalysisWorkers; worker++ {
		go func(workerID int) {
			for {
				select {
				case <-ctx.Done():
					return
				case id := <-r.queue:
					log.Printf("analysis worker %d started job %s", workerID+1, id)
					r.process(ctx, id)
				}
			}
		}(worker)
	}
}

func (r *Runner) process(ctx context.Context, id string) {
	job, err := r.store.StartAnalysis(ctx, id)
	if err != nil {
		log.Printf("job %s: %v", id, err)
		return
	}
	parsed, err := chessparser.ParsePGN(job.PGN)
	if err != nil {
		r.store.FailAnalysis(ctx, id, err)
		return
	}
	moves, err := chessparser.Positions(parsed.Game)
	if err != nil {
		r.store.FailAnalysis(ctx, id, err)
		return
	}
	fish, err := engine.Start(r.cfg.StockfishPath, r.cfg.AnalysisThreads, r.cfg.AnalysisHashMB, r.cfg.AnalysisMoveTime)
	if err != nil {
		r.store.FailAnalysis(ctx, id, err)
		return
	}
	defer fish.Close()
	moves, summary, err := engine.Analyze(ctx, fish, moves, func(p int) { _ = r.store.SetProgress(ctx, id, p) })
	if err != nil {
		r.store.FailAnalysis(ctx, id, err)
		return
	}
	if err = r.store.CompleteAnalysis(ctx, id, moves, summary); err != nil {
		r.store.FailAnalysis(ctx, id, err)
		return
	}
	log.Printf("job %s complete", id)
}
