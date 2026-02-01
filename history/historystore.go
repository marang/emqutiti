package history

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	connections "github.com/marang/emqutiti/connections"
	"github.com/marang/emqutiti/proxy"
	"github.com/sahilm/fuzzy"
	"google.golang.org/grpc"
)

var proxyAddr string

// SetProxyAddr configures the DB proxy address.
func SetProxyAddr(addr string) { proxyAddr = addr }

func addr() string {
	if proxyAddr != "" {
		return proxyAddr
	}
	return connections.LoadProxyAddr()
}

// Message holds a timestamped MQTT message with optional payload text.
type Message struct {
	Timestamp time.Time
	Topic     string
	Payload   string
	Kind      string
	Archived  bool
	Retained  bool
}

// store stores messages in memory and optionally persists them to disk.
type store struct {
	mu      sync.RWMutex
	msgs    []Message
	cl      proxy.DBProxyClient
	conn    *grpc.ClientConn
	profile string
}

// openStore opens (or creates) a persistent message index for the given profile.
// If profile is empty, "default" is used.
func openStore(profile string) (Store, error) {
	if profile == "" {
		profile = "default"
	}
	a := addr()
	if a == "" {
		return nil, nil
	}
	cl, conn, err := proxy.NewClient(a)
	if err != nil {
		return nil, err
	}
	idx := &store{cl: cl, conn: conn, profile: profile}
	resp, err := cl.Read(context.Background(), &proxy.ReadRequest{Profile: profile, Bucket: "history", Key: ""})
	if err != nil {
		conn.Close()
		return nil, err
	}
	for _, v := range resp.Values {
		var m Message
		if err := json.Unmarshal(v, &m); err != nil {
			conn.Close()
			return nil, err
		}
		idx.msgs = append(idx.msgs, m)
	}
	return idx, nil
}

// Close closes the underlying database.
func (i *store) Close() error {
	if i.conn != nil {
		return i.conn.Close()
	}
	return nil
}

// Append adds a message to the store.
func (i *store) Append(msg Message) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.msgs = append(i.msgs, msg)
	if i.cl != nil {
		key := fmt.Sprintf("%s/%020d", msg.Topic, msg.Timestamp.UnixNano())
		val, err := json.Marshal(msg)
		if err != nil {
			return err
		}
		if _, err := i.cl.Write(context.Background(), &proxy.WriteRequest{Profile: i.profile, Bucket: "history", Key: key, Value: val}); err != nil {
			return err
		}
	}
	return nil
}

// Delete removes a message with the given key from the index.
// The key should use the format "<topic>/<timestamp>" matching Add.
func (i *store) Delete(key string) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.cl != nil {
		if _, err := i.cl.Delete(context.Background(), &proxy.DeleteRequest{Profile: i.profile, Bucket: "history", Key: key}); err != nil {
			return err
		}
	}

	for idx, m := range i.msgs {
		k := fmt.Sprintf("%s/%020d", m.Topic, m.Timestamp.UnixNano())
		if k == key {
			i.msgs = append(i.msgs[:idx], i.msgs[idx+1:]...)
			break
		}
	}
	return nil
}

// Archive marks a message as archived without deleting it.
// The key should use the format "<topic>/<timestamp>" matching Add.
func (i *store) Archive(key string) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	for idx, m := range i.msgs {
		k := fmt.Sprintf("%s/%020d", m.Topic, m.Timestamp.UnixNano())
		if k == key {
			m.Archived = true
			if i.cl != nil {
				val, err := json.Marshal(m)
				if err != nil {
					return err
				}
				if _, err := i.cl.Write(context.Background(), &proxy.WriteRequest{Profile: i.profile, Bucket: "history", Key: key, Value: val}); err != nil {
					return err
				}
			}
			i.msgs[idx] = m
			return nil
		}
	}
	return fmt.Errorf("message %s not found", key)
}

// fuzzyMatchTopic reports whether the message topic fuzzy-matches any of the
// provided patterns. An empty pattern list matches all topics.
func fuzzyMatchTopic(topic string, patterns []string) bool {
	if len(patterns) == 0 {
		return true
	}
	for _, p := range patterns {
		if p == "" {
			continue
		}
		// Exact match takes priority.
		if topic == p {
			return true
		}
		// Fuzzy match: pattern should match within topic.
		matches := fuzzy.Find(p, []string{topic})
		if len(matches) > 0 {
			return true
		}
	}
	return false
}

// Search returns messages matching the provided filters. Zero timestamps
// disable the corresponding time constraints. When archived is true, only
// archived messages are returned. Topic filtering uses fuzzy matching.
func (i *store) Search(archived bool, topics []string, start, end time.Time, payload string) []Message {
	i.mu.RLock()
	defer i.mu.RUnlock()

	var out []Message
	// Filter out empty topic patterns.
	var topicPatterns []string
	for _, t := range topics {
		if t != "" {
			topicPatterns = append(topicPatterns, t)
		}
	}

	for _, m := range i.msgs {
		if m.Archived != archived {
			continue
		}
		if !fuzzyMatchTopic(m.Topic, topicPatterns) {
			continue
		}
		if !start.IsZero() && m.Timestamp.Before(start) {
			continue
		}
		if !end.IsZero() && m.Timestamp.After(end) {
			continue
		}
		if payload != "" && !strings.Contains(m.Payload, payload) {
			continue
		}
		out = append(out, m)
	}
	return out
}

// Count reports the number of stored messages. When archived is true,
// only archived messages are counted; otherwise only unarchived messages
// are included.
func (i *store) Count(archived bool) int {
	i.mu.RLock()
	defer i.mu.RUnlock()
	c := 0
	for _, m := range i.msgs {
		if m.Archived == archived {
			c++
		}
	}
	return c
}

// ParseQuery interprets a filter string in the form:
//
//	"topic=a,b start=2023-01-02T15:04:05Z end=2023-01-02T16:00 payload=foo".
//
// Fields may appear in any order and are optional. Unrecognised tokens are
// treated as payload search text.
func ParseQuery(q string) (topics []string, start, end time.Time, payload string) {
	var payloadParts []string
	for _, f := range strings.Fields(q) {
		switch {
		case strings.HasPrefix(f, "topic="):
			ts := strings.TrimPrefix(f, "topic=")
			if ts != "" {
				topics = strings.Split(ts, ",")
			}
		case strings.HasPrefix(f, "start="):
			t, err := time.Parse(time.RFC3339, strings.TrimPrefix(f, "start="))
			if err == nil {
				start = t
			}
		case strings.HasPrefix(f, "end="):
			t, err := time.Parse(time.RFC3339, strings.TrimPrefix(f, "end="))
			if err == nil {
				end = t
			}
		case strings.HasPrefix(f, "payload="):
			payloadParts = append(payloadParts, strings.TrimPrefix(f, "payload="))
		default:
			payloadParts = append(payloadParts, f)
		}
	}
	payload = strings.Join(payloadParts, " ")
	return
}
