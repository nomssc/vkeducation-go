package main

import (
	"errors"
	"fmt"
	"log"
	"sync"
)

var (
	ErrInvalidDataType = errors.New("not able to proccess this data type")
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
	wg := &sync.WaitGroup{}

	for v := range in {
		email, ok := v.(string)
		if !ok {
			log.Println(fmt.Errorf("%w: %v", ErrInvalidDataType, v))
		}

		wg.Add(1)

		go func(out chan interface{}) {
			defer wg.Done()

			user := GetUser(email)
			out <- user
		}(out)
	}

	wg.Wait()
}

func SelectMessages(in, out chan interface{}) {
	// 	in - User
	// 	out - MsgID
}

func CheckSpam(in, out chan interface{}) {
	// in - MsgID
	// out - MsgData
}

func CombineResults(in, out chan interface{}) {
	// in - MsgData
	// out - string
}
