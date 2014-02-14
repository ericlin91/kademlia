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
    "sort"
    "time"
)

// Core Kademlia type. You can put whatever state you want in this.
type Kademlia struct {
    Info *Contact
    Contact_table *Table
    Bin  map[ID][]int
    Table_ch chan int
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
    Table_ch = make(chan int)
    return ret
}

const ListSize = 10 //how many Buckets?

const alpha = 3

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
    //grab lock for the contact table
    Table_ch <- 1

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

    //release lock
    <-Table_ch
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

func (k *Kademlia) DoFindNode(remoteContact *Contact, searchKey ID) ([]FoundNode, error) {

    //fill out request, initialize request and response
    req := new(FindNodeRequest)
    req.MsgID = NewRandomID()
    req.Sender = *k.Info
    req.NodeID = searchKey
    var res FindNodeResult

    //establish connection
	address := remoteContact.Host.String() +":"+ strconv.Itoa(int(remoteContact.Port))
	client, err := rpc.DialHTTP("tcp", address)
    if err != nil {
        log.Printf("DialHTTP: ", err)
        return res.Nodes, err
    }

    //make rpc call
    err = client.Call("Kademlia.FindNode", req, &res)
    if err != nil {
        log.Printf("Call: ", err)
        return res.Nodes, err

    }

    k.Update(remoteContact)

    return res.Nodes, err
}


//Needs testing
func (k *Kademlia) DoFindValue(remoteContact *Contact, Key ID) ([]int, []FoundNode) {
	address := remoteContact.Host.String() + ":" + strconv.Itoa(int(remoteContact.Port))

	client, err := rpc.DialHTTP("tcp",address)
	if err != nil {
		log.Fatal("DialHTTP:", err)
	}

	//request
	req := new(FindValueRequest)
	req.MsgID = NewRandomID()
	req.Sender = *k.Info
	req.Key = Key

	var res FindValueResult
	err = client.Call("Kademlia.FindValue", req, &res)
	if err != nil {
		log.Fatal("Call: ", err)
	}

	k.Update(remoteContact)

	if res.Value != nil {
		fmt.Printf("No matching value!")
	} else {
		fmt.Printf("Match found!")
	}
	//Is there a way to output one or the other datatype?  
	//Not sure so just outputting both with a message to the user
	return res.Value, res.Nodes
}


func (k *Kademlia) GetContact(searchid ID) *Contact {
	var node_holder *list.Element = nil

    //get lock for contact_table
    Table_ch <- 1
	bucket_num := searchid.Xor(k.Contact_table.NodeID).PrefixLen()
	search_bucket := k.Contact_table.Buckets[bucket_num]
	for i := search_bucket.Front(); i != nil; i = i.Next() {
		
		if i.Value.(*Contact).NodeID.Equals(searchid) {
			node_holder = i
			break
		}
	}
    //release lock
    <-Table_ch

	if node_holder == nil{
		fmt.Printf("ERR\n")
        return nil
	}	else{
        return node_holder.Value.(*Contact)
	}
}

type ByDistanceFN []FoundNode

func (a ByDistanceFN) Swap(i,j int)       {a[i], a[j] = a[j], a[i]}
func (a ByDistanceFN) Len() int           {return len(a)}
func (a ByDistanceFN) Less(i, j int) bool {return a[i].NodeID.Less(a[j].NodeID)}




func removeDuplicates(nodeList []FoundNode) []FoundNode {
    resultSlice := make([]FoundNode,40)
    for i:=0; i<len(nodeList); i++ {
        found := false
        for j:=0; j<len(resultSlice); j++ {
            if nodeList[i].NodeID.Compare(resultSlice[j].NodeID) == 0 {
                found = true
            }
        }
        if found == false {
            resultSlice[i] = nodeList[i]
        }
    }
    return resultSlice
}




func (k *Kademlia) IterativeFindNode(remoteContact *Contact, searchKey ID) []FoundNode {
    // updated 20-element list containing ranked closest nodes
    short_list := make([]FoundNode,20)

    //hashmap to check if nodes have been searched yet
    checkedMap := make(map[ID]int)

    // fill short_list with first DoFindNode call and mark first contact node as checked
    checkedMap[remoteContact.NodeID] = 1
    short_list = k.DoFindNode(remoteContact, searchKey)

    // set ClosestNode to the first element of short_list
    closestNode := short_list[0]

    //changed closest node flag
    var close_node_flag int = 0
    
    //these channels are each fed into FindNodeHandler
    //make rcv_thread channel
    rcv_thread := make(chan []FoundNode)
    //make error rcv channel
    err_ch := make(chan *Contact)

    //thread return counter
    var ret_counter int = 0 

    loopFlag := true
    //loop till told to quit
    for loopFlag==true {

        select{
            //told to make a thread
            default:
                i := 0
                listFlag := true
                for listFlag == true {
                    if checkedMap[short_list[i].NodeID] == 1 { //case that node has been accessed
                        i++
                    } else if i >= len(short_list) { //went through whole shortlist
                        //break out of both inner and outer loops
                        listFlag = false
                        loopFlag = false
                    } else { //send probe to node
                        //set status as attempted to contact
                        checkedMap[short_list[i].NodeID] = 1

                        //set up call
                        var nodeToSearch Contact
                        nodeToSearch.NodeID = short_list[i].NodeID
                        nodeToSearch.Port = short_list[i].Port
                        hostconverted, err := net.LookupIP(short_list[i].IPAddr)
                        if err != nil {
                            log.Fatal("IP conversion: ", err)
                        }
                        nodeToSearch.Host = hostconverted[1]

                        //run findnode in a new thread
                        go k.FindNodeHandler(&nodeToSearch, searchKey, rcv_thread, err_ch)   
                        loopFlag = false                
                    }
                }
                time.Sleep(300 * time.Millisecond)

            //thread returns successfully
            case temp_list <- rcv_thread:
                // combine lists, remove duplicates, sort, trim to 20 elements
                new_list := make([]FoundNode,40)
                new_list = append(short_list, temp_list...)
                new_list = removeDuplicates(new_list)
                sort.Sort(ByDistanceFN(new_list))
                short_list = new_list[0:19]

                // check if closestNode is the same. If not, update 
                if short_list[0].NodeID.Compare(closestNode.NodeID) != 0 {
                    closestNode = short_list[0]
                    close_node_flag = 1
                }

                //increment return counter
                ret_counter++

            case err_node <- err_ch:
                //remove from shortlist
                for j:=0; j<len(short_list); j++ {
                    if short_list[j].NodeID.Compare(err_node.NodeID) == 0 {
                        short_list = append(short_list[:j], short_list[j+1:])
                    }
                    break
                }
                //increment return counter
                ret_counter++
        }

        //enters if every cycle, check closest node not changing condition
        if ret_counter%alpha == 0 && ret_counter!=0 {
            if close_node_flag == 0 {
                //ping everything in shortlist
                loopFlag = false
            } else { 
                //reset flag
                close_node_flag = 0
            }
        }
    }
}

func (k *Kademlia) FindNodeHandler(node *Contact, searchKey ID, rcv_thread chan []FoundNode, err_ch chan *Contact) {
    //return results of findnode through channel
    ret_list := make([]FoundNode,20)
    ret_list, err := k.DoFindNode(node, searchKey)

    if(err!=nil){
        rcv_thread <- ret_list
    }else{
        err_ch <- node
    }
}


/*
single thread iterative findnode

func (k *Kademlia) IterativeFindNode(remoteContact *Contact, searchKey ID) []FoundNode {
    // updated 20-element list containing ranked closest nodes
    short_list := make([]FoundNode,20)
    // temp_list made to hold doFindNode results without overwriting short_list
    temp_list  := make([]FoundNode,20)
    // new_list the result of combining short_list and temp_list
    new_list  := make([]FoundNode,40)
    checkedMap := make(map[ID]int)

    // fill short_list with first DoFindNode call and mark first contact node as checked
    checkedMap[remoteContact.NodeID] = 1
    short_list = k.DoFindNode(remoteContact, searchKey)

    // set ClosestNode to the first element of short_list
    closestNode := short_list[0]

    i := 0
    loopFlag := true
    for loopFlag == true {
        if checkedMap[short_list[i].NodeID] == 1 {
            i++
        } else if i >= len(short_list) {
            loopFlag = false
        } else {
            checkedMap[short_list[i].NodeID] = 1
            var nodeToSearch Contact
            nodeToSearch.NodeID = short_list[i].NodeID
            nodeToSearch.Port = short_list[i].Port
            hostconverted, err := net.LookupIP(short_list[i].IPAddr)
            if err != nil {
                log.Fatal("IP conversion: ", err)
            }
            nodeToSearch.Host = hostconverted[1]

            // temp_list holds RPC findNode call result for node short_list[i]
            temp_list = k.DoFindNode(&nodeToSearch, searchKey)

            // combine lists, remove duplicates, sort, trim to 20 elements
            new_list = Append(short_list, temp_list...)
            new_list = removeDuplicates(new_list)
            sort.Sort(ByDistanceFN(new_list))
            short_list = new_list[0:19]

            // check if closestNode is the same. If so -> exit loop
            if short_list[0].NodeID.Compare(closestNode.NodeID) == 0 {
                loopFlag = false
            } else {
                // update closestNode
                closestNode = short_list[0]
            }
        }
    }
    return short_list
}*/