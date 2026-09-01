package product

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// --------------------------------------------------------------------------
// Embedded Pure-Go RESP Redis Mock Server (Zero External Dependencies)
// --------------------------------------------------------------------------

type mockRedisServer struct {
	listener net.Listener
	mu       sync.RWMutex
	store    map[string]string
	closed   bool
}

func newMockRedisServer(t *testing.T) (*mockRedisServer, *redis.Client) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to bind mock Redis server: %v", err)
	}

	server := &mockRedisServer{
		listener: ln,
		store:    make(map[string]string),
	}

	go server.serve()

	rdb := redis.NewClient(&redis.Options{
		Addr:       ln.Addr().String(),
		MaxRetries: 0,
	})

	return server, rdb
}

func (s *mockRedisServer) serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handleConn(conn)
	}
}

func (s *mockRedisServer) handleConn(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		if len(line) == 0 {
			continue
		}

		if line[0] == '*' {
			numArgs, err := strconv.Atoi(line[1:])
			if err != nil {
				return
			}
			args := make([]string, 0, numArgs)
			for i := 0; i < numArgs; i++ {
				lenLine, err := reader.ReadString('\n')
				if err != nil {
					return
				}
				lenLine = strings.TrimRight(lenLine, "\r\n")
				if len(lenLine) == 0 || lenLine[0] != '$' {
					return
				}
				strLen, err := strconv.Atoi(lenLine[1:])
				if err != nil {
					return
				}
				buf := make([]byte, strLen+2)
				_, err = io.ReadFull(reader, buf)
				if err != nil {
					return
				}
				args = append(args, string(buf[:strLen]))
			}

			if len(args) == 0 {
				continue
			}

			cmd := strings.ToUpper(args[0])
			switch cmd {
			case "PING":
				conn.Write([]byte("+PONG\r\n"))
			case "HELLO":
				// Return unknown command so go-redis downgrades gracefully to RESP2
				conn.Write([]byte("-ERR unknown command 'HELLO'\r\n"))
			case "CLIENT":
				conn.Write([]byte("+OK\r\n"))
			case "GET":
				if len(args) < 2 {
					conn.Write([]byte("-ERR wrong number of arguments\r\n"))
					continue
				}
				s.mu.RLock()
				val, exists := s.store[args[1]]
				s.mu.RUnlock()
				if !exists {
					conn.Write([]byte("$-1\r\n"))
				} else {
					conn.Write([]byte(fmt.Sprintf("$%d\r\n%s\r\n", len(val), val)))
				}
			case "SET":
				if len(args) < 3 {
					conn.Write([]byte("-ERR wrong number of arguments\r\n"))
					continue
				}
				s.mu.Lock()
				s.store[args[1]] = args[2]
				s.mu.Unlock()
				conn.Write([]byte("+OK\r\n"))
			case "DEL":
				if len(args) < 2 {
					conn.Write([]byte("-ERR wrong number of arguments\r\n"))
					continue
				}
				count := 0
				s.mu.Lock()
				for _, key := range args[1:] {
					if _, ok := s.store[key]; ok {
						delete(s.store, key)
						count++
					}
				}
				s.mu.Unlock()
				conn.Write([]byte(fmt.Sprintf(":%d\r\n", count)))
			default:
				conn.Write([]byte("+OK\r\n"))
			}
		} else {
			// Inline command support (e.g. PING)
			parts := strings.Fields(line)
			if len(parts) > 0 && strings.ToUpper(parts[0]) == "PING" {
				conn.Write([]byte("+PONG\r\n"))
			} else {
				conn.Write([]byte("+OK\r\n"))
			}
		}
	}
}

func (s *mockRedisServer) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.closed {
		s.closed = true
		s.listener.Close()
	}
}

func (s *mockRedisServer) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.store[key]
	return val, ok
}

func (s *mockRedisServer) Set(key, val string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store[key] = val
}

// --------------------------------------------------------------------------
// 1. Redis Caching & Invalidation Adversarial Tests
// --------------------------------------------------------------------------

func TestAdversarial_RedisCacheKeyAndHit(t *testing.T) {
	server, rdb := newMockRedisServer(t)
	defer server.Close()
	defer rdb.Close()

	repo := newMockProductRepo()
	repo.products[1] = &Product{ID: 1, CategoryID: 1, Name: "Source Phone", IsActive: true}
	repo.products[2] = &Product{ID: 2, CategoryID: 1, Name: "Rec Phone 1", IsActive: true}
	repo.products[3] = &Product{ID: 3, CategoryID: 1, Name: "Rec Phone 2", IsActive: true}
	repo.products[4] = &Product{ID: 4, CategoryID: 1, Name: "Rec Phone 3", IsActive: true}
	repo.products[5] = &Product{ID: 5, CategoryID: 1, Name: "Rec Phone 4", IsActive: true}

	uc := NewUseCase(repo, rdb)
	ctx := context.Background()

	// 1. First Call: Cache Miss -> Queries repo & writes to Redis
	recs, err := uc.GetRecommendations(ctx, 1, 4)
	if err != nil {
		t.Fatalf("GetRecommendations failed: %v", err)
	}
	if len(recs) != 4 {
		t.Fatalf("Expected 4 recs, got %d", len(recs))
	}

	expectedKey := "recommendations:product:1"
	cachedVal, exists := server.Get(expectedKey)
	if !exists || cachedVal == "" {
		t.Errorf("Expected Redis key %s to be populated in cache", expectedKey)
	}

	// 2. Second Call: Cache Hit
	cachedRecs, err := uc.GetRecommendations(ctx, 1, 4)
	if err != nil {
		t.Fatalf("Cache hit call failed: %v", err)
	}
	if len(cachedRecs) != 4 {
		t.Errorf("Expected 4 cached items, got %d", len(cachedRecs))
	}
	if cachedRecs[0].ID != 2 {
		t.Errorf("Expected first cached item ID 2, got %d", cachedRecs[0].ID)
	}
}

func TestAdversarial_RedisCorruptedCacheFallback(t *testing.T) {
	server, rdb := newMockRedisServer(t)
	defer server.Close()
	defer rdb.Close()

	repo := newMockProductRepo()
	repo.products[1] = &Product{ID: 1, CategoryID: 1, Name: "Base Item", IsActive: true}
	repo.products[2] = &Product{ID: 2, CategoryID: 1, Name: "Item 2", IsActive: true}
	repo.products[3] = &Product{ID: 3, CategoryID: 1, Name: "Item 3", IsActive: true}
	repo.products[4] = &Product{ID: 4, CategoryID: 1, Name: "Item 4", IsActive: true}
	repo.products[5] = &Product{ID: 5, CategoryID: 1, Name: "Item 5", IsActive: true}

	// Inject malformed non-JSON data into Redis
	server.Set("recommendations:product:1", "{invalid_corrupted_json:[}}")

	uc := NewUseCase(repo, rdb)
	recs, err := uc.GetRecommendations(context.Background(), 1, 4)
	if err != nil {
		t.Fatalf("Corrupted cache must not cause failure; expected fallback recovery, got error: %v", err)
	}
	if len(recs) != 4 {
		t.Errorf("Expected 4 recs after corrupted cache recovery, got %d", len(recs))
	}
}

func TestAdversarial_RedisCacheInvalidationOnUpdateAndDelete(t *testing.T) {
	server, rdb := newMockRedisServer(t)
	defer server.Close()
	defer rdb.Close()

	repo := newMockProductRepo()
	cat := &Category{ID: 1, Name: "Electronics", Slug: "electronics"}
	repo.categories[1] = cat
	p := &Product{ID: 1, CategoryID: 1, Category: *cat, Name: "Original Name", Price: 100000, IsActive: true}
	repo.products[1] = p

	uc := NewUseCase(repo, rdb)
	ctx := context.Background()

	// Seed cache
	cacheKey := "recommendations:product:1"
	server.Set(cacheKey, `[{"id":2,"name":"Rec"}]`)

	// 1. Invalidate on UpdateProduct
	newName := "Updated Name"
	_, err := uc.UpdateProduct(ctx, 1, &UpdateProductRequest{Name: &newName})
	if err != nil {
		t.Fatalf("UpdateProduct failed: %v", err)
	}
	if _, exists := server.Get(cacheKey); exists {
		t.Errorf("Expected cache key %s to be invalidated after UpdateProduct", cacheKey)
	}

	// Re-seed cache
	server.Set(cacheKey, `[{"id":2,"name":"Rec"}]`)

	// 2. Invalidate on DeleteProduct
	err = uc.DeleteProduct(ctx, 1)
	if err != nil {
		t.Fatalf("DeleteProduct failed: %v", err)
	}
	if _, exists := server.Get(cacheKey); exists {
		t.Errorf("Expected cache key %s to be invalidated after DeleteProduct", cacheKey)
	}
}

// --------------------------------------------------------------------------
// 2. Fallback & Deduplication Stress Tests
// --------------------------------------------------------------------------

func TestAdversarial_AIFallbackExcludesTargetAndDeduplicates(t *testing.T) {
	repo := newMockProductRepo()
	// Target product ID = 5
	repo.products[5] = &Product{ID: 5, CategoryID: 1, Name: "Target Phone", IsActive: true}

	// Same category items (only 2 available)
	repo.products[6] = &Product{ID: 6, CategoryID: 1, Name: "Cat Phone 1", IsActive: true, Badge: "Terlaris", Rating: 4.9}
	repo.products[7] = &Product{ID: 7, CategoryID: 1, Name: "Cat Phone 2", IsActive: true, Rating: 4.5}

	// Overall items (including item 6 again, target 5, and new items 8, 9, 10)
	repo.products[8] = &Product{ID: 8, CategoryID: 2, Name: "Overall Item 1", IsActive: true, Badge: "Best Seller", Rating: 4.8}
	repo.products[9] = &Product{ID: 9, CategoryID: 3, Name: "Overall Item 2", IsActive: true, Rating: 4.7}
	repo.products[10] = &Product{ID: 10, CategoryID: 4, Name: "Overall Item 3", IsActive: true, Rating: 4.6}

	uc := NewUseCase(repo, nil)
	recs, err := uc.GetRecommendations(context.Background(), 5, 5)
	if err != nil {
		t.Fatalf("Expected graceful fallback, got error: %v", err)
	}

	if len(recs) != 5 {
		t.Fatalf("Expected exactly 5 supplemented items, got %d", len(recs))
	}

	seen := make(map[uint]bool)
	for i, r := range recs {
		if r.ID == 5 {
			t.Errorf("Target product ID 5 found in recommendations at index %d!", i)
		}
		if seen[r.ID] {
			t.Errorf("Duplicate product ID %d found in recommendations at index %d!", r.ID, i)
		}
		seen[r.ID] = true
	}
}

func TestAdversarial_RedisNilOrDisconnectedGracefulDegradation(t *testing.T) {
	repo := newMockProductRepo()
	repo.products[1] = &Product{ID: 1, CategoryID: 1, Name: "Phone", IsActive: true}
	repo.products[2] = &Product{ID: 2, CategoryID: 1, Name: "Rec 1", IsActive: true}
	repo.products[3] = &Product{ID: 3, CategoryID: 1, Name: "Rec 2", IsActive: true}
	repo.products[4] = &Product{ID: 4, CategoryID: 1, Name: "Rec 3", IsActive: true}
	repo.products[5] = &Product{ID: 5, CategoryID: 1, Name: "Rec 4", IsActive: true}

	// 1. Test when rdb is explicitly nil (e.g. running without Redis)
	ucNilRedis := NewUseCase(repo, nil)
	recs1, err1 := ucNilRedis.GetRecommendations(context.Background(), 1, 4)
	if err1 != nil {
		t.Fatalf("Expected success when rdb is nil, got %v", err1)
	}
	if len(recs1) != 4 {
		t.Errorf("Expected 4 recs, got %d", len(recs1))
	}

	// 2. Test when Redis points to unreachable address (connection refused)
	deadRdb := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:59999", // Unused port
		MaxRetries:  -1,                // No retries
		DialTimeout: 20 * time.Millisecond,
		ReadTimeout: 20 * time.Millisecond,
	})
	defer deadRdb.Close()

	ucDeadRedis := NewUseCase(repo, deadRdb)
	recs2, err2 := ucDeadRedis.GetRecommendations(context.Background(), 1, 4)
	if err2 != nil {
		t.Fatalf("Expected graceful fail-open when Redis is disconnected, got error: %v", err2)
	}
	if len(recs2) != 4 {
		t.Errorf("Expected 4 recs despite Redis disconnect, got %d", len(recs2))
	}
}

// --------------------------------------------------------------------------
// 3. Concurrency & High Load Stress Test
// --------------------------------------------------------------------------

func TestAdversarial_ConcurrentRecommendationsUnderLoad(t *testing.T) {
	server, rdb := newMockRedisServer(t)
	defer server.Close()
	defer rdb.Close()

	repo := newMockProductRepo()
	for i := uint(1); i <= 50; i++ {
		repo.products[i] = &Product{
			ID:         i,
			CategoryID: (i % 5) + 1,
			Name:       fmt.Sprintf("Product %d", i),
			Price:      float64(i * 10000),
			IsActive:   true,
			Rating:     4.5,
		}
	}

	uc := NewUseCase(repo, rdb)

	const numGoroutines = 100
	const requestsPerGoroutine = 10

	var wg sync.WaitGroup
	errCh := make(chan error, numGoroutines*requestsPerGoroutine)

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()
			for r := 0; r < requestsPerGoroutine; r++ {
				targetID := uint((routineID%10) + 1) // 10 distinct products
				limit := (r % 5) + 4                // limits between 4 and 8
				recs, err := uc.GetRecommendations(context.Background(), targetID, limit)
				if err != nil {
					errCh <- fmt.Errorf("routine %d req %d failed: %w", routineID, r, err)
					return
				}
				if len(recs) < 4 || len(recs) > 8 {
					errCh <- fmt.Errorf("routine %d req %d returned invalid length: %d", routineID, r, len(recs))
					return
				}
			}
		}(g)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatalf("Concurrent stress test error: %v", err)
	}
}
