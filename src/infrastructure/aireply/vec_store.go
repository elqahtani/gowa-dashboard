package aireply

import (
	"context"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync/atomic"

	domain "github.com/aldinokemal/go-whatsapp-web-multidevice/domains/aireply"
)

// VecStore stores embeddings in the kb_chunks.embedding BLOB column and does
// top-K cosine similarity search in Go. Doing the linear scan in pure Go (no
// sqlite-vec / pgvector) keeps the storage layer portable across SQLite and
// PostgreSQL — fine for chat-scale workloads (sub-millisecond per 1000 chunks
// at dim=1536). For larger KBs the storage abstraction can be swapped without
// touching the orchestrator.
type VecStore struct {
	db        *sql.DB
	available atomic.Bool
	dim       atomic.Int32
}

func NewVecStore(db *sql.DB) *VecStore {
	v := &VecStore{db: db}
	v.available.Store(true) // unconditionally available — no extension needed.
	return v
}

// Init sets the expected dimension. We do not allocate any structures
// up-front; the dim is used as a sanity check on Insert/Search.
func (v *VecStore) Init(dimension int) error {
	if dimension <= 0 {
		dimension = 1536
	}
	v.dim.Store(int32(dimension))
	v.available.Store(true)
	return nil
}

func (v *VecStore) Available() bool { return v.available.Load() }

// Insert updates the kb_chunks row identified by chunkID with the encoded
// embedding bytes. Must be called after the chunk row has been created.
func (v *VecStore) Insert(ctx context.Context, deviceID string, chunkID int64, embedding []float32) error {
	dim := int(v.dim.Load())
	if dim > 0 && len(embedding) != dim {
		return fmt.Errorf("embedding dim mismatch: want %d got %d", dim, len(embedding))
	}
	blob := encodeFloat32(embedding)
	_, err := v.db.ExecContext(ctx,
		`UPDATE kb_chunks SET embedding = ? WHERE id = ? AND device_id = ?`,
		blob, chunkID, deviceID)
	return err
}

func (v *VecStore) DeleteByChunkIDs(ctx context.Context, ids []int64) error {
	// chunks are owned by kb_chunks; deletion of the row removes the
	// embedding too — no separate action needed here. Provided as a no-op
	// for interface symmetry.
	_ = ctx
	_ = ids
	return nil
}

func (v *VecStore) DeleteByDevice(ctx context.Context, deviceID string) error {
	_, err := v.db.ExecContext(ctx,
		`UPDATE kb_chunks SET embedding = NULL WHERE device_id = ?`, deviceID)
	return err
}

// Search loads all embeddings for the device, computes cosine similarity in
// Go, and returns the top-K results sorted most-similar-first.
func (v *VecStore) Search(ctx context.Context, deviceID string, queryVec []float32, topK int) ([]domain.RetrievedChunk, error) {
	dim := int(v.dim.Load())
	if dim > 0 && len(queryVec) != dim {
		return nil, fmt.Errorf("query dim mismatch: want %d got %d", dim, len(queryVec))
	}
	if topK <= 0 {
		topK = 4
	}
	rows, err := v.db.QueryContext(ctx,
		`SELECT id, embedding FROM kb_chunks WHERE device_id = ? AND embedding IS NOT NULL`, deviceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	qNorm := norm(queryVec)
	if qNorm == 0 {
		return nil, errors.New("query vector is zero")
	}

	type scored struct {
		id    int64
		score float64
	}
	var all []scored
	for rows.Next() {
		var id int64
		var blob []byte
		if err := rows.Scan(&id, &blob); err != nil {
			return nil, err
		}
		vec := decodeFloat32(blob)
		if len(vec) != len(queryVec) {
			continue
		}
		sim := cosine(queryVec, vec, qNorm)
		all = append(all, scored{id: id, score: sim})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Slice(all, func(i, j int) bool { return all[i].score > all[j].score })
	if len(all) > topK {
		all = all[:topK]
	}
	out := make([]domain.RetrievedChunk, 0, len(all))
	for _, s := range all {
		out = append(out, domain.RetrievedChunk{
			Chunk:    domain.KBChunk{ID: s.id},
			Distance: 1 - s.score,
			Score:    s.score,
		})
	}
	return out, nil
}

// ============== helpers ==============

func encodeFloat32(v []float32) []byte {
	buf := make([]byte, 4*len(v))
	for i, f := range v {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

func decodeFloat32(b []byte) []float32 {
	n := len(b) / 4
	out := make([]float32, n)
	for i := 0; i < n; i++ {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return out
}

func norm(v []float32) float64 {
	var s float64
	for _, x := range v {
		s += float64(x) * float64(x)
	}
	return math.Sqrt(s)
}

func cosine(a, b []float32, normA float64) float64 {
	if normA == 0 {
		return 0
	}
	var dot, nb float64
	for i := range a {
		af, bf := float64(a[i]), float64(b[i])
		dot += af * bf
		nb += bf * bf
	}
	nrmB := math.Sqrt(nb)
	if nrmB == 0 {
		return 0
	}
	c := dot / (normA * nrmB)
	if c < -1 {
		c = -1
	} else if c > 1 {
		c = 1
	}
	return c
}

// SerializeFloat32 is exposed for tests/callers that need to stash an
// embedding into BLOB columns outside this store.
func SerializeFloat32(v []float32) ([]byte, error) {
	return encodeFloat32(v), nil
}

// suppress: keep "strings" import in case the file is later extended for
// dim-detection helpers without forcing a churn in vendored deps.
var _ = strings.TrimSpace
