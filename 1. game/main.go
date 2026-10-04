package main

func main() {
	/*
		в этой функции можно ничего не писать,
		но тогда у вас не будет работать через go run main.go
		очень круто будет сделать построчный ввод команд тут, хотя это и не требуется по заданию
	*/

	initGame()

	_ = handleCommand("some command")
}

func initGame() {
	a := NewAction()
	a.Handle("осмотреться", nil)
	a.Handle("идти", nil)
	a.Handle("применить", nil)
	a.Handle("взять", nil)
	a.Handle("надеть", nil)

	cli = NewCLI(&ConfigCLI{
		Route: a,
		Codec: NewCodec(),
	})
}
