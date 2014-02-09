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
	"fmt"
)

// Core Kademlia type. You can put whatever state you want in this.
type Kademlia struct {
    Info *Contact
    Contact_table *Table
    Bin  map[ID][]int
    //ADD CHANNELS
}

func NewKademlia(host net.IP, port uint16) *Kademlia {
    ret := new(Kademlia);
    ret.Info = new(Contact)
    ret.Info.NodeID = NewRandomID()
    ret.Info.Host = host
    ret.Info.Port = port
    ret.Contact_table = NewTable(ret.Info.NodeID)
    ret.Bin = make(map[ID][]int)
    return ret
}

const ListSize = 10 //how many Buckets?

type Table struct {
	NodeID ID
	Buckets [IDBytes*8]*list.List
}

//make a Contact_table
func NewTable(owner ID) *Table {
	Contact_table := new(Table)
	Contact_table.NodeID = owner
	//initialize Buckets
	for i := 0; i < IDBytes*8; i++ {
		Contact_table.Buckets[i] = list.New();
	}
	return Contact_table
}

   //update Contact_table
func (k *Kademlia) Update(node *Contact) {
    //first check you aren't adding yourself
    if node.NodeID.Compare(k.Contact_table.NodeID) != 0 {
        //how to initialize to nil?
        var node_holder *list.Element = nil

        //identify correct bucket
        bucket_num := node.NodeID.Xor(k.Contact_table.NodeID).PrefixLen()

        //check if node already in list
        bucket := k.Contact_table.Buckets[bucket_num]
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
            bucket.PushBack(node)
        } else if node_holder==nil && bucket.Len()==ListSize { //if new and list full, ping oldest
            //if oldest responds, do nothing
            //else drop oldest, add new to end
            //Ping(///front of list);
            //address := node.Host.String() +":"+ strconv.Itoa(int(node.Port))
            if ack := k.DoPing(node.Host, node.Port); ack == 0{
                bucket.Remove(bucket.Front())
                bucket.PushBack(node)
            }   
        
        } else{
            log.Fatal("Update failed.\n")
        }
    }
}

//func (k *Kademlia) DoPing(address string) int {
func (k *Kademlia) DoPing(rhost net.IP, port uint16) int {

    ack := 0
    address := rhost.String() +":"+ strconv.Itoa(int(port))
                    fmt.Println(address)

    client, err := rpc.DialHTTP("tcp", address)
    if err != nil {
        log.Fatal("DialHTTP: ", err)
    }

    //fill out ping
    ping := new(Ping)
    ping.MsgID = NewRandomID()
    ping.Sender = *k.Info

    var pong Pong
    err = client.Call("Kademlia.Ping", ping, &pong)
    if err != nil {
        log.Fatal("Call: ", err)
    }
    //run update with contact from pong struct
    k.Update(&pong.Sender)

    //print confirmation
    fmt.Println("ping msgID: "+ ping.MsgID.AsString())
    fmt.Println("pong msgID: "+ pong.MsgID.AsString())
    fmt.Println("pinging")
    if ping.MsgID.Compare(pong.MsgID) == 0 {
        ack = 1
    }

    return ack
}

func (k *Kademlia) DoStore(remoteContact *Contact, StoredKey ID, StoredValue []int) int {
	ack := 0
	address := remoteContact.Host.String() +":"+ strconv.Itoa(int(remoteContact.Port))

	client, err := rpc.DialHTTP("tcp", address)
    if err != nil {
        log.Fatal("DialHTTP: ", err)
    }

    //fill out request
    req := new(StoreRequest)
    req.MsgID = NewRandomID()
    req.Sender = *k.Info
    req.Key = StoredKey
    req.Value = StoredValue

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
	//ack := 0
	address := remoteContact.Host.String() +":"+ strconv.Itoa(int(remoteContact.Port))

	client, err := rpc.DialHTTP("tcp", address)
    if err != nil {
        log.Fatal("DialHTTP: ", err)
    }

    //fill out request
    req := new(FindNodeRequest)
    req.MsgID = NewRandomID()
    req.Sender = *k.Info
    req.NodeID = searchKey

    var res FindNodeResult
    err = client.Call("Kademlia.FindNode", req, &res)
    if err != nil {
        log.Fatal("Call: ", err)
    }

    k.Update(remoteContact)

    return res.Nodes
}

func (k *Kademlia) GetContact(searchid ID) *Contact {
	var node_holder *list.Element = nil
	bucket_num := searchid.Xor(k.Contact_table.NodeID).PrefixLen()
	search_bucket := k.Contact_table.Buckets[bucket_num]
	for i := search_bucket.Front(); i != nil; i = i.Next() {
		
		if i.Value.(*Contact).NodeID.Equals(searchid) {
			node_holder = i
			break
		}
	}

	if node_holder == nil{
		fmt.Printf("ERR\n")
        return nil
	}	else{
        return node_holder.Value.(*Contact)
	}
}
