package main

import (
	"fmt"
	"net"
	"os"

	"http.protocol/internal/request"
)

func main() {
	listener, err := net.Listen("tcp", ":42069")
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not listen on port 42069: %s\n", err)
		os.Exit(1)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not accept connection: %s\n", err)
			os.Exit(1)
		}

		r, err := request.RequestFromReader(conn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error parsing request: %s\n", err)
		}

		fmt.Printf("Request line:\n")
		fmt.Printf("- Method: %s\n", r.RequestLine.Method)
		fmt.Printf("- Target: %s\n", r.RequestLine.RequestTarget)
		fmt.Printf("- Version: %s\n", r.RequestLine.HttpVersion)
	}
}
