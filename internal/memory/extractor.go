package memory

import (
	"context"
	"log"
	"sync"
	"time"
)

// ExtractJob — a task for asynchronous extraction and saving of a memory.
type ExtractJob struct {
	SessionID  string
	TurnNumber int
	Content    string // "Action: X | Response: Y"
	Location   string
	ActionType string // "turn", "lore", "quest", "death"
}

// Extractor — a background worker for asynchronous embedding generation
// and saving memories to pgvector.
type Extractor struct {
	queue       chan ExtractJob
	embedClient EmbeddingProvider
	store       MemoryStore
	workers     int
	wg          sync.WaitGroup
	cancel      context.CancelFunc
}

// NewExtractor creates a new extractor with the given number of workers.
func NewExtractor(embedClient EmbeddingProvider, store MemoryStore, workers int) *Extractor {
	if workers < 1 {
		workers = 1
	}
	return &Extractor{
		queue:       make(chan ExtractJob, 100), // buffer of 100 jobs
		embedClient: embedClient,
		store:       store,
		workers:     workers,
	}
}

// Start starts N workers to process jobs.
func (e *Extractor) Start(ctx context.Context) {
	ctx, e.cancel = context.WithCancel(ctx)

	for i := 0; i < e.workers; i++ {
		e.wg.Add(1)
		go e.worker(ctx, i)
	}

	// Background retry loop for pending embeddings every 30 seconds
	e.wg.Add(1)
	go e.retryLoop(ctx)

	log.Printf("[Extractor] Started %d workers", e.workers)
}

// Submit sends a job to the queue (non-blocking).
func (e *Extractor) Submit(job ExtractJob) {
	select {
	case e.queue <- job:
	default:
		// Queue is full — save without embedding
		log.Printf("[Extractor] Queue full, saving without embedding: session=%s turn=%d",
			job.SessionID, job.TurnNumber)
		go e.saveWithoutEmbedding(context.Background(), job)
	}
}

// Stop stops all workers and waits for completion.
func (e *Extractor) Stop() {
	if e.cancel != nil {
		e.cancel()
	}
	e.wg.Wait()
	log.Println("[Extractor] Stopped")
}

// worker processes jobs from the queue.
func (e *Extractor) worker(ctx context.Context, id int) {
	defer e.wg.Done()

	for {
		select {
		case job := <-e.queue:
			e.processJob(ctx, job)
		case <-ctx.Done():
			// Process remaining jobs in the queue
			for {
				select {
				case job := <-e.queue:
					e.processJob(context.Background(), job)
				default:
					return
				}
			}
		}
	}
}

// processJob processes a single job: generates an embedding and saves.
func (e *Extractor) processJob(ctx context.Context, job ExtractJob) {
	mem := &Memory{
		SessionID:  job.SessionID,
		TurnNumber: job.TurnNumber,
		Content:    job.Content,
		Location:   job.Location,
		ActionType: job.ActionType,
		CreatedAt:  time.Now(),
	}

	// Try to get the embedding
	embedding, err := e.embedClient.Embed(ctx, job.Content)
	if err != nil {
		log.Printf("[Extractor] Embedding error (session=%s turn=%d): %v — saving without vector",
			job.SessionID, job.TurnNumber, err)
		mem.Embedded = false
	} else {
		mem.Embedding = embedding
		mem.Embedded = true
	}

	// Save to PostgreSQL
	if err := e.store.InsertMemory(ctx, mem); err != nil {
		log.Printf("[Extractor] Save error (session=%s turn=%d): %v",
			job.SessionID, job.TurnNumber, err)
	}
}

// saveWithoutEmbedding saves a record without a vector (fallback).
func (e *Extractor) saveWithoutEmbedding(ctx context.Context, job ExtractJob) {
	mem := &Memory{
		SessionID:  job.SessionID,
		TurnNumber: job.TurnNumber,
		Content:    job.Content,
		Location:   job.Location,
		ActionType: job.ActionType,
		Embedded:   false,
		CreatedAt:  time.Now(),
	}
	if err := e.store.InsertMemory(ctx, mem); err != nil {
		log.Printf("[Extractor] Fallback save error: %v", err)
	}
}

// retryLoop periodically recalculates embeddings for records with embedded=false.
func (e *Extractor) retryLoop(ctx context.Context) {
	defer e.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.retryPending(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// retryPending recalculates embeddings for pending records.
func (e *Extractor) retryPending(ctx context.Context) {
	pending, err := e.store.PendingEmbeddings(ctx, 10)
	if err != nil {
		log.Printf("[Extractor] Error loading pending: %v", err)
		return
	}

	if len(pending) == 0 {
		return
	}

	log.Printf("[Extractor] Recalculating %d pending embeddings", len(pending))

	for _, m := range pending {
		select {
		case <-ctx.Done():
			return
		default:
		}

		embedding, err := e.embedClient.Embed(ctx, m.Content)
		if err != nil {
			log.Printf("[Extractor] Retry embedding error (id=%d): %v", m.ID, err)
			continue
		}

		if err := e.store.UpdateEmbedding(ctx, m.ID, embedding); err != nil {
			log.Printf("[Extractor] Retry update error (id=%d): %v", m.ID, err)
		}
	}
}
