package main

import "github.com/tigerbot/vue"

type Data struct {
	Message string
}

func main() {
	vue.New(
		vue.El("#app"),
		vue.Template("<p>{{ Message }}</p>"),
		vue.Data(Data{Message: "Hello WebAssembly!"}),
	)

	select {}
}
