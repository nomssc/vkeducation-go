package main

import (
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
)

// RunPipeline runs cmds as a pipeline: the output channel of every command is
// connected to the input channel of the next one, like the stages of a unix
// pipe (cmd1 | cmd2 | cmd3).
//
// Every command runs in its own goroutine, so data flows through the stages
// without being accumulated. The first command receives an already closed
// input channel. After a command returns, its output channel is closed, which
// lets the next command finish its "for range in" loop. RunPipeline returns
// only when all commands have finished.
func RunPipeline(cmds ...cmd) {
	wg := &sync.WaitGroup{}

	in := make(chan interface{})
	close(in)

	for _, c := range cmds {
		out := make(chan interface{})

		wg.Add(1)

		go func(c cmd, in chan interface{}, out chan interface{}) {
			defer wg.Done()
			defer close(out)

			c(in, out)
		}(c, in, out)

		in = out
	}

	wg.Wait()
}

// Map is a goroutine-safe set of uint64 keys.
type Map struct {
	store map[uint64]struct{}

	mu sync.Mutex
}

// NewMap returns an empty, ready-to-use Map.
func NewMap() *Map {
	return &Map{
		store: map[uint64]struct{}{},
	}
}

// Set adds key to the set. It is safe for concurrent use.
func (m *Map) Set(key uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.store[key] = struct{}{}
}

// Get reports whether key is in the set. It is safe for concurrent use.
func (m *Map) Get(key uint64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok := m.store[key]

	return ok
}

// SelectUsers is the first pipeline stage. It reads user e-mails (strings) from
// in, resolves each of them to a User with GetUser, and sends every unique user
// to out.
//
// GetUser is slow (one second per call), so it is invoked concurrently for all
// e-mails. Several e-mails may be aliases of the same user; such users are
// deduplicated by ID and passed on only once. Values that are not non-empty
// strings are skipped.
func SelectUsers(in, out chan interface{}) {
	uniqueUsers := NewMap()

	wg := &sync.WaitGroup{}

	for v := range in {
		email, ok := v.(string)
		if !ok || email == "" {
			continue
		}

		wg.Add(1)

		go func(email string, out chan interface{}) {
			defer wg.Done()

			user := GetUser(email)

			ok := uniqueUsers.Get(user.ID)
			if !ok {
				uniqueUsers.Set(user.ID)
				out <- user
			}
		}(email, out)
	}

	wg.Wait()
}

// SelectMessages is the second pipeline stage. It reads User values from in,
// collects them into batches of at most GetMessagesMaxUsersBatch users, and
// requests the messages of each batch with a single GetMessages call. Every
// resulting MsgID is sent to out.
//
// Batches are processed concurrently. When in is closed, the last, possibly
// incomplete batch is flushed. Values that are not Users, or Users with a zero
// ID or an empty e-mail, are skipped.
func SelectMessages(in, out chan interface{}) {
	users := make([]User, 0, GetMessagesMaxUsersBatch)

	wg := &sync.WaitGroup{}

	for {
		v, running := <-in
		if !running {
			if len(users) > 0 {
				wg.Add(1)
				go proccessBatch(wg, users, out)
			}

			break
		}

		user, ok := v.(User)
		if !ok || user.ID == 0 || user.Email == "" {
			continue
		}

		users = append(users, user)

		if len(users) == GetMessagesMaxUsersBatch {
			wg.Add(1)
			go proccessBatch(wg, users, out)

			users = make([]User, 0, GetMessagesMaxUsersBatch)
		}
	}

	wg.Wait()
}

// proccessBatch fetches the messages of users with one GetMessages call and
// sends every returned MsgID to out. It marks wg as done when it returns.
// The error of GetMessages is ignored: on error no messages are returned.
func proccessBatch(wg *sync.WaitGroup, users []User, out chan interface{}) {
	defer wg.Done()

	msgs, _ := GetMessages(users...)

	for _, msg := range msgs {
		out <- msg
	}
}

// RateLimiter counts the requests that are currently in flight. It is safe for
// concurrent use.
type RateLimiter struct {
	count atomic.Int64
}

// Load returns the current counter value.
func (r *RateLimiter) Load() int {
	return int(r.count.Load())
}

// Add changes the counter by delta.
func (r *RateLimiter) Add(delta int) {
	r.count.Add(int64(delta))
}

// Clear resets the counter to zero.
func (r *RateLimiter) Clear() {
	r.count.Store(0)
}

// CheckSpam is the third pipeline stage. It reads message IDs (MsgID) from in,
// checks each of them with HasSpam, and sends a MsgData with the result to out.
//
// The anti-spam service rejects more than HasSpamMaxAsyncRequests parallel
// requests, so messages are checked in waves: once that many requests have been
// started, CheckSpam waits for all of them to finish before starting the next
// wave. Values that are not MsgIDs, or are zero, are skipped. The error of
// HasSpam is ignored.
func CheckSpam(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	rl := &RateLimiter{}

	for v := range in {
		msgID, ok := v.(MsgID)
		if !ok || msgID == 0 {
			continue
		}

		wg.Add(1)
		rl.Add(1)

		go func(msgID MsgID, out chan interface{}) {
			defer wg.Done()

			msgData := MsgData{ID: msgID}

			spam, _ := HasSpam(msgID)
			msgData.HasSpam = spam

			out <- msgData
		}(msgID, out)

		if rl.Load() == HasSpamMaxAsyncRequests {
			wg.Wait()
			rl.Clear()
		}
	}

	wg.Wait()
	rl.Clear()
}

// CombineResults is the last pipeline stage. It collects all MsgData values
// from in until the channel is closed, sorts them, and sends one string per
// message to out in the form "<has_spam> <msg_id>", for example
// "true 17696166526272393238".
//
// Messages with spam come first; within each group messages are ordered by ID
// in ascending order. Values that are not MsgData, or have a zero ID, are
// skipped.
func CombineResults(in, out chan interface{}) {
	msgs := make([]MsgData, 0)

	for v := range in {
		msgData, ok := v.(MsgData)
		if !ok || msgData.ID == 0 {
			continue
		}

		msgs = append(msgs, msgData)
	}

	slices.SortFunc(msgs, func(a, b MsgData) int {
		if a.HasSpam != b.HasSpam {
			if a.HasSpam {
				return -1
			}
			return 1
		}

		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}

		return 0
	})

	for _, msg := range msgs {
		out <- fmt.Sprintf("%v %v", msg.HasSpam, msg.ID)
	}
}
