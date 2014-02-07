package kademlia
// Contains the core kademlia type. In addition to core state, this type serves
// as a receiver for the RPC methods, which is required by that package.
// blah blah
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
    table *Table
    bin  map[ID][]byte
}

func NewKademlia(host net.IP, port uint16) *Kademlia {
    ret := new(Kademlia);
    ret.info = new(Contact)
    ret.info.NodeID = NewRandomID()
    ret.info.Host = host
    ret.info.Port = port
    ret.table = NewTable(ret.info.NodeID)
    ret.bin = make(map[ID][]byte)
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
func (k *Kademlia) Update(node *Contact) {
	//first check you aren't adding yourself
	if node.NodeID.Compare(k.table.NodeID) != 0 {
		//how to initialize to nil?
		var node_holder *list.Element = nil

		//identify correct bucket
		bucket_num := node.NodeID.Xor(k.table.NodeID).PrefixLen()

		//check if node already in list
		bucket := k.table.buckets[bucket_num]
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
			if ack := k.DoPing(node.Host, node.Port); ack == 0{
				bucket.Remove(bucket.Front())
				bucket.PushBack(node_holder)
			}	
		
		} else{
			log.Fatal("Update failed.\n")
		}
	}
}

func (k *Kademlia) DoPing(rhost net.IP, port uint16) int {
	ack := 0
	address := rhost.String() +":"+ strconv.Itoa(int(port))

	client, err := rpc.DialHTTP("tcp", address)
    if err != nil {
        log.Fatal("DialHTTP: ", err)
    }

    //fill out ping
    ping := new(Ping)
    ping.MsgID = NewRandomID()
    ping.Sender = *k.info

    var pong Pong
    err = client.Call("Kademlia.Ping", ping, &pong)
    if err != nil {
        log.Fatal("Call: ", err)
    }
    //run update with contact from pong struct
    k.Update(&pong.Sender)

    //print confirmation
    log.Printf("ping msgID: %s\n", ping.MsgID.AsString())
    log.Printf("pong msgID: %s\n", pong.MsgID.AsString())

    if ping.MsgID.Compare(pong.MsgID) == 0 {
    	ack = 1
    }

    return ack
}

func (k *Kademlia) DoStore(remoteContact *Contact, Key ID, Value []byte) int {
	ack := 0
	address := remoteContact.Host.String() +":"+ strconv.Itoa(int(remoteContact.Port))

	client, err := rpc.DialHTTP("tcp", address)
    if err != nil {
        log.Fatal("DialHTTP: ", err)
    }

    //fill out request
    req := new(StoreRequest)
    req.MsgID = NewRandomID()
    req.Sender = *k.info

    var res StoreResult
    err = client.Call("Kademlia.Store", req, &res)
    if err != nil {
        log.Fatal("Call: ", err)
    }

    //print confirmation
    log.Printf("req msgID: %s\n", req.MsgID.AsString())
    log.Printf("res msgID: %s\n", res.MsgID.AsString())

    if req.MsgID.Compare(res.MsgID) == 0 {
    	ack = 1
    	k.Update(remoteContact)
    }

    return ack
}

func (k *Kademlia) DoFindNode(remoteContact *Contact, searchKey ID) []FoundNode {
	ack := 0
	address := remoteContact.Host.String() +":"+ strconv.Itoa(int(remoteContact.Port))

	client, err := rpc.DialHTTP("tcp", address)
    if err != nil {
        log.Fatal("DialHTTP: ", err)
    }

    //fill out request
    req := new(FindNodeRequest)
    req.MsgID = NewRandomID()
    req.Sender = *k.info
    req.NodeID = searchKey

    var res FindNodeResult
    err = client.Call("Kademlia.FindNode", req, &res)
    if err != nil {
        log.Fatal("Call: ", err)
    }

    k.Update(remoteContact)

    return res.Nodes
}