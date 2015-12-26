package main

import (
  "log"
  "fmt"
  "net"
  "bufio"
  "time"
  "os"
)

/*
    Get external IP and listen on it
    Dial up address of other node
*/

type client chan<- string

var (
  leave = make(chan client)
  enter = make(chan client)
  messages = make(chan string)
)

func writeMessages(ch chan string, conn net.Conn) {
  for {
    msg := <-ch
    fmt.Fprintf(conn, msg)
  }
}

func broadcast() {
  connections := make(map[client]bool)
  for {
    select {
    case msg := <-messages:
      for cli := range connections {
        cli <- msg
      }
    case client := <-enter:
      connections[client] = true
    case client := <-leave:
      delete(connections, client)
      close(client)
    default:
      // do nothing
    }
  }
}

func getName(conn net.Conn, scanner *bufio.Scanner) string {
  fmt.Fprintln(conn, "What's your name?")
  scanner.Scan()
  if err := scanner.Err(); err != nil {
    return conn.RemoteAddr().String()
  }
  return scanner.Text()
}

func getPrefix(name string) string {
  return time.Now().Format("15:04") + "\t" + name + ": "
}

func handleConn(conn net.Conn) {
  defer conn.Close()
  scanner := bufio.NewScanner(conn)

  name := getName(conn, scanner)

  ch := make(chan string)
  enter <- ch

  messages <- getPrefix(name) + " has joined the room\n"
  fmt.Println(getPrefix(name), "has joined the room")

  go writeMessages(ch, conn)
  for scanner.Scan() {
    messages <- getPrefix(name) + scanner.Text() + "\n"
  }
  if err := scanner.Err(); err != nil {
    log.Println(err)
  }

  messages <- getPrefix(name) + " has left the room\n"
  fmt.Println(getPrefix(name), " has left the room")
  leave <- ch
  conn.Close()
}


func getExternalIP() (string, error) {
  return "localhost" + ":" + "8080", nil
}

func main() {
  address := "localhost:8080"
  if len(os.Args) >= 2 {
    address = os.Args[1]
  }

  ip, err := getExternalIP()

  listener, err := net.Listen("tcp", address)

  fmt.Println(ip)

  fmt.Println("listening on", listener.Addr())
  if err != nil {
    log.Fatal(err)
  }
  go broadcast()
  for {
    conn, err := listener.Accept()
    if err != nil {
      log.Println(err)
      continue
    }
    go handleConn(conn)
  }
}
