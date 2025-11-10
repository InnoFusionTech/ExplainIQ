package pool

import (
	"context"
	"sync"
	"time"
)

// Pool represents a connection pool
type Pool struct {
	factory      func() (interface{}, error)
	close        func(interface{}) error
	maxSize      int
	idleTimeout  time.Duration
	items        chan *poolItem
	mu           sync.Mutex
	totalCreated int
}

type poolItem struct {
	conn        interface{}
	lastUsed    time.Time
}

// NewPool creates a new connection pool
func NewPool(factory func() (interface{}, error), close func(interface{}) error, maxSize int, idleTimeout time.Duration) *Pool {
	pool := &Pool{
		factory:     factory,
		close:       close,
		maxSize:     maxSize,
		idleTimeout: idleTimeout,
		items:       make(chan *poolItem, maxSize),
	}

	// Start cleanup goroutine
	go pool.cleanup()

	return pool
}

// Get gets a connection from the pool
func (p *Pool) Get(ctx context.Context) (interface{}, error) {
	// Try to get from pool
	select {
	case item := <-p.items:
		// Check if idle timeout exceeded
		if time.Since(item.lastUsed) > p.idleTimeout {
			// Close expired connection
			if p.close != nil {
				p.close(item.conn)
			}
			p.mu.Lock()
			p.totalCreated--
			p.mu.Unlock()
			// Create new connection
			return p.createNew()
		}
		return item.conn, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Pool empty, create new
		p.mu.Lock()
		if p.totalCreated >= p.maxSize {
			p.mu.Unlock()
			// Wait for connection or timeout
			select {
			case item := <-p.items:
				return item.conn, nil
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(5 * time.Second):
				return nil, context.DeadlineExceeded
			}
		}
		p.totalCreated++
		p.mu.Unlock()
		return p.createNew()
	}
}

// Put returns a connection to the pool
func (p *Pool) Put(conn interface{}) {
	item := &poolItem{
		conn:     conn,
		lastUsed: time.Now(),
	}

	select {
	case p.items <- item:
		// Successfully returned to pool
	default:
		// Pool full, close connection
		if p.close != nil {
			p.close(conn)
		}
		p.mu.Lock()
		p.totalCreated--
		p.mu.Unlock()
	}
}

// createNew creates a new connection
func (p *Pool) createNew() (interface{}, error) {
	return p.factory()
}

// cleanup periodically removes idle connections
func (p *Pool) cleanup() {
	ticker := time.NewTicker(p.idleTimeout / 2)
	defer ticker.Stop()

	for range ticker.C {
		p.mu.Lock()
		size := len(p.items)
		p.mu.Unlock()

		// Clean up idle connections
		for i := 0; i < size; i++ {
			select {
			case item := <-p.items:
				if time.Since(item.lastUsed) > p.idleTimeout {
					if p.close != nil {
						p.close(item.conn)
					}
					p.mu.Lock()
					p.totalCreated--
					p.mu.Unlock()
				} else {
					// Return to pool
					select {
					case p.items <- item:
					default:
						if p.close != nil {
							p.close(item.conn)
						}
					}
				}
			default:
				break
			}
		}
	}
}

// Close closes all connections in the pool
func (p *Pool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	close(p.items)
	for item := range p.items {
		if p.close != nil {
			p.close(item.conn)
		}
	}
}











