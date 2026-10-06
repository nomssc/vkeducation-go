package main

import (
	"sync"
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
	uniqueUsers := make(map[uint64]struct{})

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

			_, ok := uniqueUsers[user.ID]
			if !ok {
				uniqueUsers[user.ID] = struct{}{}
				out <- user
			}
		}(out)
	}

	wg.Wait()
}

func SelectMessages(in, out chan interface{}) {
	users := make([]User, GetMessagesMaxUsersBatch)
	i := 0

	wg := &sync.WaitGroup{}

	for v := range in {
		user, ok := v.(User)
		if !ok || user.ID == 0 || user.Email == "" {
			continue
		}

		if i < GetMessagesMaxUsersBatch-1 {
			users[i] = user
			i++
			continue
		}

		dummy := make([]User, 0, len(users))
		copy(dummy, users)

		i = 0
		wg.Add(1)

		go func(users []User, out chan interface{}) {
			defer wg.Done()

			msgs, _ := GetMessages(users...)

			for _, msg := range msgs {
				out <- msg
			}
		}(dummy, out)
	}

	wg.Wait()
}

func CheckSpam(in, out chan interface{}) {
	// in - MsgID
	// out - MsgData
}

func CombineResults(in, out chan interface{}) {
	// in - MsgData
	// out - string
}
