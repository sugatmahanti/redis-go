package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

func main() {
	fmt.Println("Logs from your program will appear here!")

	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		buff := make([]byte, 256)
		redisMap := make(map[string]string, 10)
		go func() {
			for {
				n, err := conn.Read(buff)
				if err == io.EOF {
					break
				}
				if err != nil {
					fmt.Println("Error reading from connection: ", err.Error())
					os.Exit(1)
				}

				if n > 0 {

					input := string(buff[:n])

					msg := extractRESPString(input)
					fmt.Println(msg)
					response := ""
					switch strings.ToUpper(msg[0]) {
					case "ECHO":
						response = fmt.Sprintf("$%d\r\n%s\r\n", len(msg[1]), msg[1])
					case "PING":
						response = fmt.Sprintf("+%s\r\n", "PONG")
					case "GET":
						response = fmt.Sprintf("$%d\r\n%s\r\n", len(redisMap[msg[1]]), redisMap[msg[1]])
					case "SET":
						redisMap[msg[1]] = msg[2]
						response = fmt.Sprintf("+%s\r\n", "OK")
					}

					conn.Write([]byte(response))
				}
			}
		}()
	}
}

func extractRESPString(input string) []string {

	chars := strings.Split(input, "\r\n")

	if chars[0][0] == '*' {
		cmd := []string{}
		for i := 0; i < len(chars); i++ {
			if len(chars[i]) > 0 && chars[i][0] == '$' {
				i++
				if i < len(chars) {
					cmd = append(cmd, chars[i])
				}
			}
		}

		return cmd

	}

	return []string{}
}
