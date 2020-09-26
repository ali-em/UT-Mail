package main

import (
	"fmt"

	_ "github.com/ali-em/UT-Mail/mail"
	"github.com/ali-em/UT-Mail/telegram"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in main", r)
		}
		fmt.Println("Closing App")
	}()
	telegram.Setup()
}
