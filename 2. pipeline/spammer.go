package main

import (
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
)

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

func SelectUsers(in, out chan interface{}) {
	uniqueUsers := sync.Map{}

	wg := &sync.WaitGroup{}

	for v := range in {
		email, ok := v.(string)
		if !ok {
			continue
		}

		wg.Add(1)

		go func(out chan interface{}) {
			defer wg.Done()

			user := GetUser(email)

			_, ok := uniqueUsers.Load(user.ID)
			if !ok {
				uniqueUsers.Store(user.ID, struct{}{})
				out <- user
			}
		}(out)
	}

	wg.Wait()
}

func SelectMessages(in, out chan interface{}) {
	users := make([]User, 0, GetMessagesMaxUsersBatch)

	wg := &sync.WaitGroup{}

	for {
		v, end := <-in
		if end {
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

func proccessBatch(wg *sync.WaitGroup, users []User, out chan interface{}) {
	defer wg.Done()

	msgs, _ := GetMessages(users...)

	for _, msg := range msgs {
		out <- msg
	}
}

type RateLimiter struct {
	count atomic.Int64
}

func (r *RateLimiter) Load() int {
	return int(r.count.Load())
}

func (r *RateLimiter) Add(delta int) {
	r.count.Add(int64(delta))
}

func (r *RateLimiter) Clear() {
	r.count.Store(0)
}

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
