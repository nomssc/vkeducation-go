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

func CheckSpam(in, out chan interface{}) {
	// in - MsgID
	// out - MsgData
}

func CombineResults(in, out chan interface{}) {
	// in - MsgData
	// out - string
}
