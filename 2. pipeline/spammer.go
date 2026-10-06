package main

import "sync"

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
	// 	in - string
	// 	out - User
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
