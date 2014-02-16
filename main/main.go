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
    kadem := kademlia.NewKademlia(ip[0], uint16(port))

    rpc.Register(kadem)
    rpc.HandleHTTP()
    l, err := net.Listen("tcp", listenStr)
    if err != nil {
        log.Fatal("Listen: ", err)
    }
    log.Printf("ping msgIDa:\n")

    // Serve forever.
    go http.Serve(l, nil)
    log.Printf("ping msgIDb:\n")

    //run bucketaccess
    go kadem.BucketAccess()

    // Confirm our server is up with a PING request and then exit.
    // Your code should loop forever, reading instructions from stdin and
    // printing their results to stdout. See README.txt for more details.
    client, err := rpc.DialHTTP("tcp", firstPeerStr)
    if err != nil {
        log.Fatal("DialHTTP: ", err)
    }
        log.Printf("ping msgIDc:\n")

    ping := new(kademlia.Ping)
    ping.MsgID = kademlia.NewRandomID()
    var pong kademlia.Pong
    err = client.Call("Kademlia.Ping", ping, &pong)
        log.Printf("ping msgIDd:\n")

    if err != nil {
        log.Fatal("Call: ", err)
    }

    log.Printf("ping msgID: %s\n", ping.MsgID.AsString())
    log.Printf("pong msgID: %s\n", pong.MsgID.AsString())

    //kadem.DoPing(ip, 7890)

    //command line interface
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
                    log.Printf("Ping setup error: ", err)
                }

                ip_index := 0
                if strings.Contains(cmd_arr[1], "localhost") {
                    ip_index = 1
                } else {
                    ip_index = 0
                }
                err = kadem.DoPing(host[ip_index], uint16(port))                    
            } else {
                id_to_ping,err := kademlia.FromString(cmd_arr[1])
                if err != nil {
                    log.Printf("Ping setup error: ", err)
                }
                contact_to_ping := kadem.GetContact(id_to_ping)
                err = kadem.DoPing(contact_to_ping.Host, contact_to_ping.Port)
            }

        case "store":
            input_id, err := kademlia.FromString(cmd_arr[1])
            key_id, err := kademlia.FromString(cmd_arr[2])
            if err != nil {
                log.Printf("Store setup error: ", err)
            }
            store_loc := kadem.GetContact(input_id)

            var teststore []byte = nil

            err = kadem.DoStore(store_loc, key_id, teststore)

        case "find_node":
            input_id, err := kademlia.FromString(cmd_arr[1])
            key_id, err := kademlia.FromString(cmd_arr[2])
            if err != nil {
                log.Printf("FindNode setup error: ", err)
            }
            contact := kadem.GetContact(input_id)

            node_list, err := kadem.DoFindNode(contact, key_id)
            fmt.Println(node_list[0].NodeID.AsString())       

        case "find_value":
            /*input_id, err := kademlia.FromString(cmd_arr[1])
            key_id, err := kademlia.FromString(cmd_arr[2])
            if err != nil {
                log.Fatal("Find Value setup error: ", err)
            }
            id_contact := kadem.GetContact(input_id)
            value_return, nodes_return := kadem.DoFindValue(id_contact, key_id)
            if nodes_return != nil {
                //fmt.Printf(nodes_return)
            } else {
                //fmt.Printf(value_return)
            }*/

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
            fmt.Printf("fnode = %s\n", fnode.NodeID.AsString())
            fmt.Println("IP Address: ", fnode.Host.String())
            fmt.Println("Port: ", fnode.Port)

        case "iterativeStore":
            /*key_id, err := kademlia.FromString(cmd_arr[1])
            value_id, err := kademlia.FromString(cmd_arr[2])
            if err != nil {
                log.Fatal("Iterative Store setup error: ", err)
            }
            last_node := kadem.IterativeStore(key_id, value_id)*/

        case "iterativeFindNode":
            input_id, err := kademlia.FromString(cmd_arr[1])
            if err != nil {
                log.Printf("Iterative Find Node setup error: ", err)
            }
            found_nodes, err := kadem.IterativeFindNode(input_id)
            
            for j := 0; j < len(found_nodes); j++ {
                fmt.Println(found_nodes[j].NodeID.AsString())
            }


        case "iterativeFindValue":
            /*key_id, err := kademlia.FromString(cmd_arr[1])
            if err != nil {
                log.Printf("Iterative Find Value setup error :", err)
            }
            value_node, value := kadem.iterativeFindValue(key_id)
            fmt.Printf("%v %v\n", value_node, value)*/
        }
    }
}

