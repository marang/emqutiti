package proxy

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/dgraph-io/badger/v4"
	"github.com/marang/emqutiti/internal/files"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/stats"
)

// Proxy runs a gRPC server mediating database access.
type Proxy struct {
	UnimplementedDBProxyServer
	srv *grpc.Server
	lis net.Listener

	mu  sync.Mutex
	dbs map[string]*badger.DB

	reads   uint64
	writes  uint64
	deletes uint64
	clients int64
}

var (
	proxyMu      sync.Mutex
	proxyRunning bool
)

const (
	// valueLogFileSizeBytes caps BadgerDB value log files to keep the initial
	// on-disk footprint around 10MB while still allowing the database to grow
	// as new segments are allocated.
	valueLogFileSizeBytes = 10 * 1024 * 1024
)

// StartProxy starts the gRPC proxy on addr. Only one may run at a time.
func StartProxy(addr string) (*Proxy, error) {
	proxyMu.Lock()
	defer proxyMu.Unlock()
	if proxyRunning {
		return nil, fmt.Errorf("proxy already running")
	}
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	p := &Proxy{dbs: make(map[string]*badger.DB)}
	p.srv = grpc.NewServer(grpc.StatsHandler(&proxyStats{p: p}))
	p.lis = lis
	RegisterDBProxyServer(p.srv, p)
	proxyRunning = true
	go p.srv.Serve(lis)
	return p, nil
}

// Stop stops the proxy and closes all database handles.
func (p *Proxy) Stop() {
	p.srv.GracefulStop()
	p.mu.Lock()
	for _, db := range p.dbs {
		db.Close()
	}
	p.dbs = make(map[string]*badger.DB)
	p.mu.Unlock()
	proxyMu.Lock()
	proxyRunning = false
	proxyMu.Unlock()
}

// Addr returns the listening address.
func (p *Proxy) Addr() string { return p.lis.Addr().String() }

func (p *Proxy) dbKey(profile, bucket string) string {
	return profile + "|" + bucket
}

func (p *Proxy) getDB(profile, bucket string) (*badger.DB, error) {
	key := p.dbKey(profile, bucket)
	p.mu.Lock()
	defer p.mu.Unlock()
	if db, ok := p.dbs[key]; ok {
		return db, nil
	}
	if profile == "" {
		profile = "default"
	}
	path := filepath.Join(files.DataDir(profile), bucket)
	if err := files.EnsureDir(path); err != nil {
		return nil, err
	}
	opts := badger.DefaultOptions(path).
		WithLogger(nil).
		WithValueLogFileSize(valueLogFileSizeBytes)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	p.dbs[key] = db
	return db, nil
}

// Write stores a key/value pair.
func (p *Proxy) Write(ctx context.Context, req *WriteRequest) (*WriteResponse, error) {
	db, err := p.getDB(req.GetProfile(), req.GetBucket())
	if err != nil {
		return nil, err
	}
	err = db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(req.GetKey()), req.GetValue())
	})
	if err != nil {
		return nil, err
	}
	atomic.AddUint64(&p.writes, 1)
	return &WriteResponse{}, nil
}

// Read returns all values with the given key prefix.
func (p *Proxy) Read(ctx context.Context, req *ReadRequest) (*ReadResponse, error) {
	db, err := p.getDB(req.GetProfile(), req.GetBucket())
	if err != nil {
		return nil, err
	}
	var vals [][]byte
	err = db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte(req.GetKey())
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			v, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			vals = append(vals, v)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	atomic.AddUint64(&p.reads, 1)
	return &ReadResponse{Values: vals}, nil
}

// Delete removes all keys with the given prefix.
func (p *Proxy) Delete(ctx context.Context, req *DeleteRequest) (*DeleteResponse, error) {
	db, err := p.getDB(req.GetProfile(), req.GetBucket())
	if err != nil {
		return nil, err
	}
	err = db.Update(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte(req.GetKey())
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			if err := txn.Delete(it.Item().KeyCopy(nil)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	atomic.AddUint64(&p.deletes, 1)
	return &DeleteResponse{}, nil
}

// Status reports usage and database size metrics.
func (p *Proxy) Status(ctx context.Context, _ *StatusRequest) (*StatusResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	infos := make([]*DBInfo, 0, len(p.dbs))
	for k, db := range p.dbs {
		lsm, vlog := db.Size()
		parts := strings.SplitN(k, "|", 2)
		prof, bucket := parts[0], parts[1]
		var entries uint64
		if err := db.View(func(txn *badger.Txn) error {
			it := txn.NewIterator(badger.DefaultIteratorOptions)
			defer it.Close()
			for it.Rewind(); it.Valid(); it.Next() {
				entries++
			}
			return nil
		}); err != nil {
			return nil, err
		}
		infos = append(infos, &DBInfo{
			Profile: prof,
			Bucket:  bucket,
			Size:    uint64(lsm + vlog),
			Entries: entries,
		})
	}
	return &StatusResponse{
		Dbs:     infos,
		Reads:   atomic.LoadUint64(&p.reads),
		Writes:  atomic.LoadUint64(&p.writes),
		Deletes: atomic.LoadUint64(&p.deletes),
		Clients: atomic.LoadInt64(&p.clients),
	}, nil
}

type proxyStats struct{ p *Proxy }

func (ps *proxyStats) TagRPC(ctx context.Context, _ *stats.RPCTagInfo) context.Context   { return ctx }
func (ps *proxyStats) HandleRPC(context.Context, stats.RPCStats)                         {}
func (ps *proxyStats) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context { return ctx }
func (ps *proxyStats) HandleConn(ctx context.Context, s stats.ConnStats) {
	switch s.(type) {
	case *stats.ConnBegin:
		atomic.AddInt64(&ps.p.clients, 1)
	case *stats.ConnEnd:
		atomic.AddInt64(&ps.p.clients, -1)
	}
}

// NewClient returns a client connected to the proxy at addr. Used in tests.
func NewClient(addr string) (DBProxyClient, *grpc.ClientConn, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	return NewDBProxyClient(conn), conn, nil
}
