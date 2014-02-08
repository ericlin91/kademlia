package main

import (
    "flag"
    "fmt"
    "log"
    "math/rand"
    "net"
    "net/http"
    "net/rpc"
    "time"
    "strings"
    "strconv"
    "bufio"
    "os"
)

import (
    "kademlia/kademlia"
)


func main() {
    // By default, Go seeds its RNG with 1. This would cause every program to
    // generate the same sequence of IDs.
    rand.Seed(time.Now().UnixNano())

    // Get the bind and connect connection strings from command-line arguments.
    flag.Parse()
    args := flag.Args()
    if len(args) != 2 {
        log.Fatal("Must be invoked with exactly two arguments!\n")
    }
    listenStr := args[0]
    firstPeerStr := args[1]

    //catch ip and port of listener so we can pass it to others
    ip_and_port := strings.Split(listenStr,":")
    ip := net.ParseIP(ip_and_port[0])
    port,err := strconv.ParseUint(ip_and_port[1], 0, 16)

    fmt.Printf("kademlia starting up!\n")
    kadem := kademlia.NewKademlia(ip, uint16(port))

    rpc.Register(kadem)
    rpc.HandleHTTP()
    l, err := net.Listen("tcp", listenStr)
    if err != nil {
        log.Fatal("Listen: ", err)
    }

    // Serve forever.
    go http.Serve(l, nil)

    // Confirm our server is up with a PING request and then exit.
    // Your code should loop forever, reading instructions from stdin and
    // printing their results to stdout. See README.txt for more details.
    client, err := rpc.DialHTTP("tcp", firstPeerStr)
    if err != nil {
        log.Fatal("DialHTTP: ", err)
    }
    ping := new(kademlia.Ping)
    ping.MsgID = kademlia.NewRandomID()
    var pong kademlia.Pong
    err = client.Call("Kademlia.Ping", ping, &pong)
    if err != nil {
        log.Fatal("Call: ", err)
    }

    log.Printf("ping msgID: %s\n", ping.MsgID.AsString())
    log.Printf("pong msgID: %s\n", pong.MsgID.AsString())

    //kadem.DoPing(ip, 7890)

    for {
        //bio := bufio.NewReader(os.Stdin)
        scanner := bufio.NewScanner(bufio.NewReader(os.Stdin))
        scanner.Split(bufio.ScanLines)
        scanner.Scan()
        //log.Printf("%s\n",scanner.Text())
        cmd_arr := strings.Split(scanner.Text(), " ")
        switch cmd_arr[0] {
        case "ping":

        case "store":

        case "find_node":

        case "find_value":

        case "whoami":
            fmt.Println(kadem.Info.NodeID.AsString())
        case "local_find_value":
        
        case "get_contact":

        case "iterativeStore":

        case "iterativeFindNode":

        case "iterativeFindValue":

        }
        //read from input
        //bufio.Reader
        //switch on cmd
        //case "ping"
        //  doPing()        
    }
}

