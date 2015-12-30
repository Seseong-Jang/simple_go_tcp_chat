package main

import (
	"bufio"
	"bytes"
	"container/list"
	"fmt"
	"net"
	"strings"
	"time"
)

const (
	LOGIN = "1"
	CHAT  = "2"
)

type Client struct {
	conn net.Conn
	read chan string
	quit chan int
	name string
}

var clientlist *list.List

func main() {
	clientlist = list.New()

	ln, err := net.Listen("tcp", ":5000")
	if err != nil {
		handleError(nil, err, "server listen error..")
	}
	defer ln.Close()

	for {
		// waiting connection
		conn, err := ln.Accept()
		if err != nil {
			handleError(conn, err, "server accept error..")
		}

		go handleConnection(conn)
	}
}

func handleError(conn net.Conn, err error, errmsg string) {
	if conn != nil {
		conn.Close()
	}
	fmt.Println(err)
	fmt.Println(errmsg)
}

func handleConnection(conn net.Conn) {
	read := make(chan string)
	quit := make(chan int)

	client := &Client{conn, read, quit, "unknown"}

	go handleClient(client)

	fmt.Printf("remote Addr = %s\n", conn.RemoteAddr().String())
}

func handleClient(client *Client) {
	for {
		select {
		case msg := <-client.read:
			if strings.Contains(msg, "[B]") {
				sendToAllClients(client.name, msg)
			} else {
				sendToClient(client, client.name, msg)
			}

		case <-client.quit:
			fmt.Println("disconnect client")
			client.conn.Close()
			client.deleteFromList()
			return

		default:
			go recvFromClient(client)
			time.Sleep(1000 * time.Millisecond)
		}
	}
}

func recvFromClient(client *Client) {
	recvmsg, err := bufio.NewReader(client.conn).ReadString('\n')
	if err != nil {
		handleError(client.conn, err, "string read error..")
		client.quit <- 0
		return
	}

	strmsgs := strings.Split(recvmsg, "|")

	switch strmsgs[0] {
	case LOGIN:
		client.name = strings.TrimSpace(strmsgs[1])
		if !client.dupUserCheck() {
			handleError(client.conn, nil, "duplicate user!!"+client.name)
			client.quit <- 0
			return
		}
		fmt.Printf("\nhello = %s\n", client.name)
		clientlist.PushBack(*client)

	case CHAT:
		fmt.Printf("\nrecv message = %s\n", strmsgs[1])
		client.read <- strmsgs[1]
	}
}

func sendToClient(client *Client, sender string, msg string) {
	var buffer bytes.Buffer
	buffer.WriteString("[")
	buffer.WriteString(sender)
	buffer.WriteString("] ")
	buffer.WriteString(msg)

	fmt.Printf("client = %s ==> %s", client.name, buffer.String())

	fmt.Fprintf(client.conn, "%s", buffer.String())
}

func sendToAllClients(sender string, msg string) {
	fmt.Printf("broad cast message = %s", msg)
	for e := clientlist.Front(); e != nil; e = e.Next() {
		c := e.Value.(Client)
		sendToClient(&c, sender, msg)
	}
}

func (client *Client) deleteFromList() {
	for e := clientlist.Front(); e != nil; e = e.Next() {
		c := e.Value.(Client)
		if client.conn == c.conn {
			clientlist.Remove(e)
		}
	}
}

func (client *Client) dupUserCheck() bool {
	for e := clientlist.Front(); e != nil; e = e.Next() {
		c := e.Value.(Client)
		if strings.Compare(client.name, c.name) == 0 {
			return false
		}
	}

	return true
}
