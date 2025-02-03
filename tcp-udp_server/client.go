package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: client <protocol> <port>")
		return
	}

	protocol := os.Args[1]
	port := os.Args[2]

	conn, err := net.Dial(protocol, port)

	if err != nil {
		fmt.Println("Error: ", err)
	}
	defer conn.Close()

	fmt.Fprintf(conn, "Hello World\n")

}
