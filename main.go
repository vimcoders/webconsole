package main

import (
	"net/http"

	_ "github.com/vimcoders/webconsole/generator"
)

func main() {
	http.ListenAndServe(":8001", nil)
}
