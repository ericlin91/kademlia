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
    ip, err := net.LookupIP(ip_and_port[0])
    port,err := strconv.ParseUint(ip_and_port[1], 0, 16)

    fmt.Printf("kademlia starting up!\n")
    kadem := kademlia.NewKademlia(ip[1], uint16(port))

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
        cmd_arr := strings.Split(scanner.Text(), " ")

        switch cmd_arr[0] {
        case "ping":
            //case with host:port
            if host_port := strings.Split(cmd_arr[1], ":"); host_port[0] != cmd_arr[1] {
                port, err := strconv.ParseUint(host_port[1], 0, 16)
                host, err := net.LookupIP(host_port[0])

                if err != nil {
                    log.Fatal("ParseUint: ", err)
                }

                if kadem.DoPing(host[1], uint16(port) ) == 0 {
                    fmt.Println("Ping failed.")
                }
            }

        case "store":
            input_id, err := kademlia.FromString(cmd_arr[1])
            key_id, err := kademlia.FromString(cmd_arr[2])
            if err != nil {
                log.Fatal("Input error: ", err)
            }
            store_loc := kadem.GetContact(input_id)

            teststore := []int{2,99}

            kadem.DoStore(store_loc, key_id, teststore)

        case "find_node":
            input_id, err := kademlia.FromString(cmd_arr[1])
            key_id, err := kademlia.FromString(cmd_arr[2])
            if err != nil {
                log.Fatal("Input error: ", err)
            }
            contact := kadem.GetContact(input_id)

            kadem.DoFindNode(contact, key_id)       

        case "find_value":

        case "whoami":
            fmt.Println(kadem.Info.NodeID.AsString())

        case "local_find_value":
            input_id, err := kademlia.FromString(cmd_arr[1])
            if err != nil {
                log.Fatal("Contact: ", err)
            }
            map_data := kadem.Bin[input_id]
            if map_data == nil {
                fmt.Println("ERR")
            } else {
                fmt.Println(map_data)
            }

        case "get_contact":
            input_id, err := kademlia.FromString(cmd_arr[1])
            if err != nil {
                log.Fatal("Contact: ", err)
            }
            fnode := kadem.GetContact(input_id)
            fmt.Println("IP Address: ", fnode.Host.String())
            fmt.Println("Port: ", fnode.Port)

        case "iterativeStore":

        case "iterativeFindNode":

        case "iterativeFindValue":

        }
      
    }
}

