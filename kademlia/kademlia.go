package kademlia
// Contains the core kademlia type. In addition to core state, this type serves
// as a receiver for the RPC methods, which is required by that package.

import(
	"container/list"
	"strconv"
	"net"
	"net/rpc"
	"log"
)

// Core Kademlia type. You can put whatever state you want in this.
type Kademlia struct {
    info *Contact
    rtable *Table
}

func NewKademlia(host net.IP, port uint16) *Kademlia {
    ret := new(Kademlia);
    ret.info = new(Contact)
    ret.info.NodeID = NewRandomID()
    ret.info.Host = host
    ret.info.Port = port
    ret.rtable = NewTable(ret.info.NodeID)
    return ret
}

const ListSize = 10 //how many buckets?

type Table struct {
	NodeID ID
	buckets [IDBytes*8]*list.List
}

//make a table
func NewTable(owner ID) *Table {
	table := new(Table)
	table.NodeID = owner
	//initialize buckets
	for i := 0; i < IDBytes*8; i++ {
		table.buckets[i] = list.New();
	}
	return table
}

//update table
func (table *Table) Update(node *Contact) {
	//how to initialize to nil?
	var node_holder *list.Element = nil

	//identify correct bucket
	bucket_num := node.NodeID.Xor(table.NodeID).PrefixLen()

	//check if node already in list
	bucket := table.buckets[bucket_num]
	for i := bucket.Front(); i != nil; i = i.Next() {
		if i.Value.(*Contact).NodeID.Equals(node.NodeID) {
			node_holder = i
			break
		}
	}

	//if old, move to end
	if node_holder!=nil {
		bucket.MoveToBack(node_holder)
	} else if node_holder==nil && bucket.Len()!=ListSize { //if new and list not full, add to end
		bucket.PushBack(node_holder)
	} else if node_holder==nil && bucket.Len()==ListSize { //if new and list full, ping oldest
		
		//if oldest responds, do nothing
		//else drop oldest, add new to end
		//Ping(///front of list);
		// if { //no pong
		// 	bucket.Remove(bucket.Front())
		// 	bucket.PushBack(node_holder)
		// }
	} else{
		log.Fatal("Update failed.\n")
	}
}

func (k *Kademlia) DoPing(rhost net.IP, port uint16) {
	address := rhost.String() +":"+ strconv.Itoa(int(port))

	client, err := rpc.DialHTTP("tcp", address)
    if err != nil {
        log.Fatal("DialHTTP: ", err)
    }
    ping := new(Ping)
    ping.MsgID = NewRandomID()
    //fill out rest of ping/contact struct
    var pong Pong
    err = client.Call("Kademlia.Ping", ping, &pong)
    if err != nil {
        log.Fatal("Call: ", err)
    }
    //run update with contact from pong struct

    log.Printf("ping msgID: %s\n", ping.MsgID.AsString())
    log.Printf("pong msgID: %s\n", pong.MsgID.AsString())
}