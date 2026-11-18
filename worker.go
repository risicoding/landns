package landns

import "sync"

type Worker struct {
	limit int
	wg    sync.WaitGroup
	mu    sync.Mutex
	ch    chan func()
}

func NewWorker(limit int) *Worker {
	ch := make(chan func())
	var wg sync.WaitGroup
	go func() {
		for range limit {
			wg.Add(1)
			defer wg.Done()

			go execute(&ch)
		}

	}()
	return &Worker{
		limit: limit,
	}
}

func execute(ch *chan func()) {
	for function := range *ch {
		function()
	}
}

func (w *Worker) Do(function func()) {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		function()
	}()
}

func (w *Worker) Wait() {
	w.wg.Wait()
}

func (w *Worker) Lock() {
	w.mu.Lock()
}

func (w *Worker) Unlock() {
	w.mu.Unlock()
}
