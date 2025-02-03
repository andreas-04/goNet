package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func handleConnection(c net.Conn) {
	defer c.Close()
	reader := bufio.NewReader(c)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			if err.Error() != "EOF" {
				fmt.Println("Error reading from connection:", err)
			}
			break
		}
		fmt.Print("Message received: ", message)
	}

}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: server <protocol> <port>")
		return
	}

	protocol := os.Args[1]
	port := os.Args[2]

	ln, err := net.Listen(protocol, port)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go handleConnection(conn)
	}
}
