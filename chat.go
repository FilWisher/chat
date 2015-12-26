/*
    dial client
    
    listen for incoming    
    
    take input from command line and write to clientOut
    
    when message received, listen to addr if not already

    serialize and deserialize message
*/
package main

import (
  "log"
  "encoding/gob"
  "bytes"
  "fmt"
  "net"
  "io"
  "bufio"
  "os"
  "strings"
)

type Peer struct {
  ChannOut chan Message
  ChannIn chan Message
  Addr string
}

type Message struct {
  Body string
  Addr string
}

var (
  arrivers = make(chan Peer)
  leavers = make(chan Peer)
  messages = make(chan Message)
)

func (m Message) GobEncode() ([]byte, error) {
  var b bytes.Buffer
  fmt.Fprintln(&b, m.Body, m.Addr)
  return b.Bytes(), nil
}

func (m *Message) GobDecode(data []byte) error {
  b := bytes.NewBuffer(data)
  _, err := fmt.Fscanln(b, &m.Body, &m.Addr)
  return err
}

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
    /* TODO: this logic makes spaces not correct */
    /*      likely because of decoding and encoding */
    if err != nil {
      if err == io.EOF {
        end <- true
        break
      }
    }
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
  peer.Addr = conn.RemoteAddr().String()

  arrivers <- peer

  go writeTo(peer.ChannOut, conn)
  go readFrom(peer.ChannIn, conn, end)

  <-end

  fmt.Println("closin now")
  leavers <- peer
  conn.Close()
}

func managePeers() {
  peers := make(map[Peer]string)

  for {
    select {
      case peer := <-arrivers:
        peers[peer] = peer.Addr
      case peer := <-leavers:
        close(peer.ChannOut)
        close(peer.ChannIn)
        delete(peers, peer)
      case msg := <-messages:
        for peer := range peers {
          peer.ChannOut <- msg
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
  default:
    messages <- msg
  }
}

func init() {
  log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {

  l, err := net.Listen("tcp", "localhost" + ":0")
  if err != nil {
    log.Fatal(err)
  }
  fmt.Printf("listening on %s\n", l.Addr().String())
  go managePeers()
  go listen(l)

  scanner := bufio.NewScanner(os.Stdin)

  for scanner.Scan() {
    if err = scanner.Err(); err != nil {
      fmt.Println(err)
      continue
    }
    msg := Message{scanner.Text(), l.Addr().String()}
    go runCommand(msg)
  }
}
