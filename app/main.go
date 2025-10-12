package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	fmt.Println("Logs from your program will appear here!")

	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}

	redisMap := make(RedisMap, 10)
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go func(c net.Conn) {
			defer c.Close()

			buff := make([]byte, 256)
			for {
				n, err := c.Read(buff)
				if err == io.EOF {
					return
				}
				if err != nil {
					fmt.Println("Error reading from connection: ", err.Error())
					return
				}
				if n == 0 {
					continue
				}

				msg := extractRESPString(string(buff[:n]))
				if len(msg) == 0 {
					continue
				}

				var response string
				switch strings.ToUpper(msg[0]) {
				case "ECHO":
					response = fmt.Sprintf("$%d\r\n%s\r\n", len(msg[1]), msg[1])
				case "PING":
					response = "+PONG\r\n"
				case "GET":
					val, found := redisMap.getValue(msg[1])
					if found {
						response = fmt.Sprintf("$%d\r\n%s\r\n", len(val), val)
					} else {
						response = "$-1\r\n"
					}
				case "SET":
					if len(msg) >= 5 {
						expiryTime, _ := strconv.Atoi(msg[4])
						if strings.ToUpper(msg[3]) == "PX" || strings.ToUpper(msg[3]) == "EX" {
							redisMap.setValueWithExpiry(msg[1], msg[2], msg[3], expiryTime)
						}
					} else {
						redisMap.setValue(msg[1], msg[2])
					}
					response = "+OK\r\n"
				default:
					response = "-ERR unknown command\r\n"
				}

				if _, err := c.Write([]byte(response)); err != nil {
					fmt.Println("Error writing response: ", err.Error())
					return
				}
			}
		}(conn)
	}
}

type RedisMap map[string]RedisMapValue

func (rm RedisMap) setValueWithExpiry(key string, value string, expiryType string, expiryTime int) {
	duration := time.Duration(expiryTime)
	if strings.ToUpper(expiryType) == "PX" {
		duration *= time.Millisecond
	} else if strings.ToUpper(expiryType) == "EX" {
		duration *= time.Second
	}
	expiry := time.Now().Add(duration)
	rm[key] = RedisMapValue{
		Value:  value,
		Expiry: &expiry,
	}
}

func (rm RedisMap) setValue(key string, value string) {
	rm[key] = RedisMapValue{
		Value: value,
	}
}

func (rm RedisMap) getValue(key string) (string, bool) {
	if entry, ok := rm[key]; ok {
		if entry.Expiry != nil {
			if entry.Expiry.After(time.Now()) {
				return entry.Value, true
			} else {
				delete(rm, key)
				return "", false
			}
		} else if entry.Expiry == nil {
			return entry.Value, true
		}
	}
	return "", false
}

type RedisMapValue struct {
	Value  string
	Expiry *time.Time
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
