package main

import (
  "log"
  "encoding/gob"
  "fmt"
  "net"
  "io"
  "bufio"
  "os"
  "strings"
  "math/rand"
  "time"
  "github.com/filwisher/utils/extip"
)

func GUID(n int) string {
  var letters = []rune("abcdef0123456789")
  b := make([]rune, n)
  for i := range b {
    b[i] = letters[rand.Intn(len(letters))]
  }
  return string(b)
}

type Peer struct {
  ChannOut chan Message
  ChannIn chan Message
  Addr string
}

type Message struct {
  Body string
  Addr string
  ID string
}

var (
  arrivers = make(chan Peer)
  leavers = make(chan Peer)
  messages = make(chan Message)
  peers = make(map[string]Peer)
  received = make(map[string]bool)
)

var id string

func writeTo(ch chan Message, conn net.Conn) {
  enc := gob.NewEncoder(conn)
  for {
    msg := <-ch
    enc.Encode(msg)
  }
}

func readFrom(ch chan Message, conn net.Conn, end chan bool) {
  dec := gob.NewDecoder(conn)
  var msg Message
  for {
    err := dec.Decode(&msg)
    if err != nil {
      if err == io.EOF {
        end <- true
        break
      }
      log.Println(err)
      continue
    }
    _, ok := received[msg.ID]
    if ok {
      return
    }
    select {
    case messages <- msg:
      //
    default:
      //
    }
    received[msg.ID] = true

    fmt.Println(msg.Addr + ": " + msg.Body)
  }
}

func connect(address string) {

  fmt.Printf("Connecting to %s\n", address)
  end := make(chan bool)

  conn, err := net.Dial("tcp", address)
  if err != nil {
    log.Println(err)
    return
  }

  var peer Peer
  peer.ChannOut = make(chan Message)
  peer.ChannIn = make(chan Message)
  peer.Addr = address

  arrivers <- peer

  go writeTo(peer.ChannOut, conn)
  go readFrom(peer.ChannIn, conn, end)

  <-end

  leavers <- peer
  conn.Close()
}

func managePeers() {

  for {
    select {
      case peer := <-arrivers:
        peers[peer.Addr] = peer
        fmt.Println("peers::::")
        fmt.Println(peers)
      case peer := <-leavers:
        close(peer.ChannOut)
        close(peer.ChannIn)
        delete(peers, peer.Addr)
      case msg := <-messages:
        _, ok := peers[msg.Addr]
        if !ok && msg.Addr != id {
          fmt.Println("not exists")
          go connect(msg.Addr)
        }
        for _, peer := range peers {
          if peer.Addr != msg.Addr {
            peer.ChannOut <- msg
          }
        }
    }
  }
}

func handleConnection(conn net.Conn) {

  end := make(chan bool)

  var peer Peer
  peer.ChannOut = make(chan Message, 2)
  peer.ChannIn = make(chan Message, 2)
  peer.Addr = conn.RemoteAddr().String()

  fmt.Printf("now connected to %s\n", conn.RemoteAddr().String())

  arrivers <- peer

  go writeTo(peer.ChannOut, conn)
  go readFrom(peer.ChannIn, conn, end)

  <-end

  leavers <- peer
  conn.Close()

}

func listen(l net.Listener) {
  for {
    conn, err := l.Accept()
    if err != nil {
      log.Println(err)
      continue
    }
    go handleConnection(conn)
  }
}

func isLongEnough(args []string, length int) bool {
  return len(args) >= length
}

/* define irc-like commands */
func runCommand(msg Message) {
  args := strings.Split(msg.Body, " ")
  switch args[0] {
  case "/connect":
    if !isLongEnough(args, 2) {
      log.Println("Not enough arguments")
      return
    }
    go connect(args[1])
  case "/info":
    fmt.Println(msg.Addr)
  case "/peers":
    for addr, peer := range(peers) {
      fmt.Printf(addr)
      fmt.Println(peer)
    }
  default:
    messages <- msg
  }
}

func init() {
  log.SetFlags(log.LstdFlags | log.Lshortfile)
  rand.Seed(time.Now().UnixNano())
}

func main() {

  l, err := net.Listen("tcp", extip.Extip() + ":0")
  if err != nil {
    log.Fatal(err)
  }
  id = l.Addr().String()
  fmt.Printf("listening on %s\n", id)
  go managePeers()
  go listen(l)

  scanner := bufio.NewScanner(os.Stdin)

  for scanner.Scan() {
    if err = scanner.Err(); err != nil {
      fmt.Println(err)
      continue
    }
    msg := Message{scanner.Text(), id, GUID(9)}
    received[msg.ID] = true
    go runCommand(msg)
  }
}
