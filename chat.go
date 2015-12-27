// TODO: migrate to messages and decode/encode

package main

import (
  "net"
  "fmt"
  "bufio"
  "os"
  "encoding/gob"
  "strings"
  "math/rand"
  "time"
)

var (
  ID, Name string
)

type Message struct {
  Text, ID, Sender, Name string
}

var (
  send = make(chan Message)
  conns = make(map[*gob.Encoder]bool)
  had = make(map[string]bool)
)

func id(n int) string {
  letters := []rune("abcdef0123456789")
  b := make([]rune, n)
  for i := range b {
    b[i] = letters[rand.Intn(len(letters))]
  }
  return string(b)
}

func broadcastMessage(text string) {
  msg := Message{text, id(7), ID, Name}
  send <- msg
}

func connect (address string) {
  conn, err := net.Dial("tcp", address)
  if err != nil {
    fmt.Println("couldn't connect")
    return
  }
  fmt.Printf("connecting to %s\n", address)
  handle(conn)
}

func runCommand(command string) {
  args := strings.Split(command, " ")
  switch args[0] {
  case "/connect":
    if len(args) < 2 {
      fmt.Println("Not enough args")
      return
    }
    go connect(args[1])
  default:
    fmt.Println("I didn't recognize that")
  }
}

func readStdin() {
  sc := bufio.NewScanner(os.Stdin)
  for sc.Scan() {
    if err := sc.Err(); err != nil {
      continue
    }
    input := sc.Text()
    if string(input[0]) == "/" {
      go runCommand(input)
    } else {
      go broadcastMessage(input)
    }
  }
}

func getMessage(dec *gob.Decoder, fn func (Message)) {
  var msg Message
  for {
    err := dec.Decode(&msg)
    if err != nil {
      fmt.Println("err")
      return
    }
    _, ok := had[msg.ID]
    had[msg.ID] = true
    if !ok && msg.Sender != ID {
      fn(msg)
    }
  }
}

func writeToConns() {
  for {
    msg := <-send
    for enc := range(conns) {
      enc.Encode(msg)
    }
  }
}

func handle(conn net.Conn) {
  enc := gob.NewEncoder(conn)
  dec := gob.NewDecoder(conn)
  conns[enc] = true
  getMessage(dec, func (msg Message) {
    fmt.Printf("%s: %s\n", msg.Name, msg.Text)
    send <- msg
  })
  delete(conns, enc)
}

func init() {
  rand.Seed(time.Now().UnixNano())
  ID = id(10)
}

func getName() string {
  fmt.Println("What is your name?")
  scanner := bufio.NewScanner(os.Stdin)
  scanner.Scan()
  if err := scanner.Err(); err != nil {
    panic("name not working")
  }
  return scanner.Text()
}

func main() {

  l, _ := net.Listen("tcp", "localhost:0")
  fmt.Printf("listening on %s\n", l.Addr().String())

  Name = getName()

  go readStdin()
  go writeToConns()

  for {
    conn, err := l.Accept()
    if err != nil {
      continue
    }
    fmt.Printf("received conn from %s\n", conn.RemoteAddr().String())
    go handle(conn)
  }
}
