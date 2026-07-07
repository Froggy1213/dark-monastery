package memory

import (
	"context"
	"log"
	"sync"
	"time"
)

// ExtractJob — задание для асинхронного извлечения и сохранения воспоминания.
type ExtractJob struct {
	SessionID  string
	TurnNumber int
	Content    string // "Действие: X | Ответ: Y"
	Location   string
	ActionType string // "turn", "lore", "quest", "death"
}

// Extractor — фоновый воркер для асинхронной генерации эмбеддингов
// и сохранения воспоминаний в pgvector.
type Extractor struct {
	queue       chan ExtractJob
	embedClient EmbeddingProvider
	store       MemoryStore
	workers     int
	wg          sync.WaitGroup
	cancel      context.CancelFunc
}

// NewExtractor создаёт новый экстрактор с заданным количеством воркеров.
func NewExtractor(embedClient EmbeddingProvider, store MemoryStore, workers int) *Extractor {
	if workers < 1 {
		workers = 1
	}
	return &Extractor{
		queue:       make(chan ExtractJob, 100), // буфер на 100 заданий
		embedClient: embedClient,
		store:       store,
		workers:     workers,
	}
}

// Start запускает N воркеров для обработки заданий.
func (e *Extractor) Start(ctx context.Context) {
	ctx, e.cancel = context.WithCancel(ctx)

	for i := 0; i < e.workers; i++ {
		e.wg.Add(1)
		go e.worker(ctx, i)
	}

	// Фоновый пересчёт pending embeddings каждые 30 секунд
	e.wg.Add(1)
	go e.retryLoop(ctx)

	log.Printf("[Extractor] Запущено %d воркеров", e.workers)
}

// Submit отправляет задание в очередь (неблокирующий).
func (e *Extractor) Submit(job ExtractJob) {
	select {
	case e.queue <- job:
	default:
		// Очередь полна — сохраняем без эмбеддинга
		log.Printf("[Extractor] Очередь полна, сохраняем без эмбеддинга: session=%s turn=%d",
			job.SessionID, job.TurnNumber)
		go e.saveWithoutEmbedding(context.Background(), job)
	}
}

// Stop останавливает все воркеры и ждёт завершения.
func (e *Extractor) Stop() {
	if e.cancel != nil {
		e.cancel()
	}
	e.wg.Wait()
	log.Println("[Extractor] Остановлен")
}

// worker обрабатывает задания из очереди.
func (e *Extractor) worker(ctx context.Context, id int) {
	defer e.wg.Done()

	for {
		select {
		case job := <-e.queue:
			e.processJob(ctx, job)
		case <-ctx.Done():
			// Дообрабатываем оставшиеся в очереди
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

// processJob обрабатывает одно задание: генерирует эмбеддинг и сохраняет.
func (e *Extractor) processJob(ctx context.Context, job ExtractJob) {
	mem := &Memory{
		SessionID:  job.SessionID,
		TurnNumber: job.TurnNumber,
		Content:    job.Content,
		Location:   job.Location,
		ActionType: job.ActionType,
		CreatedAt:  time.Now(),
	}

	// Пытаемся получить эмбеддинг
	embedding, err := e.embedClient.Embed(ctx, job.Content)
	if err != nil {
		log.Printf("[Extractor] Ошибка embedding (session=%s turn=%d): %v — сохраняем без вектора",
			job.SessionID, job.TurnNumber, err)
		mem.Embedded = false
	} else {
		mem.Embedding = embedding
		mem.Embedded = true
	}

	// Сохраняем в PostgreSQL
	if err := e.store.InsertMemory(ctx, mem); err != nil {
		log.Printf("[Extractor] Ошибка сохранения (session=%s turn=%d): %v",
			job.SessionID, job.TurnNumber, err)
	}
}

// saveWithoutEmbedding сохраняет запись без вектора (fallback).
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
		log.Printf("[Extractor] Ошибка fallback сохранения: %v", err)
	}
}

// retryLoop периодически пересчитывает эмбеддинги для записей с embedded=false.
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

// retryPending пересчитывает эмбеддинги для pending записей.
func (e *Extractor) retryPending(ctx context.Context) {
	pending, err := e.store.PendingEmbeddings(ctx, 10)
	if err != nil {
		log.Printf("[Extractor] Ошибка загрузки pending: %v", err)
		return
	}

	if len(pending) == 0 {
		return
	}

	log.Printf("[Extractor] Пересчитываем %d pending эмбеддингов", len(pending))

	for _, m := range pending {
		select {
		case <-ctx.Done():
			return
		default:
		}

		embedding, err := e.embedClient.Embed(ctx, m.Content)
		if err != nil {
			log.Printf("[Extractor] Ошибка retry embedding (id=%d): %v", m.ID, err)
			continue
		}

		if err := e.store.UpdateEmbedding(ctx, m.ID, embedding); err != nil {
			log.Printf("[Extractor] Ошибка retry update (id=%d): %v", m.ID, err)
		}
	}
}
