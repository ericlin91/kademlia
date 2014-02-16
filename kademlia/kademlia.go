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
    "errors"
)

// Core Kademlia type. You can put whatever state you want in this.
type Kademlia struct {
    Info *Contact
    Contact_table *Table
    Bin  map[ID][]byte
    Update_ch chan *Contact
    Update_err_ch chan error
    FindNode_in_ch chan FindNodeRequest
    FindNode_out_ch chan []*Contact
    GetContact_in chan ID
    GetContact_out chan *Contact
}

func NewKademlia(host net.IP, port uint16) *Kademlia {
    ret := new(Kademlia);
    ret.Info = new(Contact)
    ret.Info.NodeID = NewRandomID()
    ret.Info.Host = host
    ret.Info.Port = port
    ret.Contact_table = NewTable(ret.Info.NodeID)
    ret.Bin = make(map[ID][]byte)
    ret.Update_ch = make(chan *Contact)
    ret.Update_err_ch = make(chan error)
    ret.FindNode_in_ch = make(chan FindNodeRequest)
    ret.FindNode_out_ch = make(chan []*Contact)
    ret.GetContact_in = make(chan ID)
    ret.GetContact_out = make(chan *Contact)

    return ret
}

const ListSize = 10 //how many Buckets?

const Alpha = 3

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
func (k *Kademlia) UpdateHandler(node *Contact) error {

    //first check you aren't adding yourself
    if node.NodeID.Compare(k.Contact_table.NodeID) != 0 {
        
        //holder for contact from list
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
            if err := k.DoPing(node.Host, node.Port); err == nil{
                bucket.Remove(bucket.Front())
                bucket.PushBack(node)
            }         
        } else{
            err := errors.New("Update failed.\n")
            //<-k.Table_ch
            return err
        }
    }
    return nil
}

func (k *Kademlia) Update(node *Contact) error {
    k.Update_ch <- node
    err := <- k.Update_err_ch
    return err
}


//concurrency k-bucket access handler
func (k *Kademlia) BucketAccess() {
    for{
        select{
            case newUpdate:= <- k.Update_ch: //pull contact from update channel
                err := k.UpdateHandler(newUpdate)            
                k.Update_err_ch <- err
                
            case newReq := <- k.FindNode_in_ch: //pull something from findnodew channel
                bucket_slice := k.ReturnLocalBuckets(newReq)
                k.FindNode_out_ch <- bucket_slice

            case searchid := <- k.GetContact_in:

                // takes NodeID, returns corresponding Contact Pointer
                var node_holder *list.Element = nil
                bucket_num := searchid.Xor(k.Contact_table.NodeID).PrefixLen()
                search_bucket := k.Contact_table.Buckets[bucket_num]
                for i := search_bucket.Front(); i != nil; i = i.Next() {
                    
                    if i.Value.(*Contact).NodeID.Equals(searchid) {
                        node_holder = i
                        break
                    }
                }
                //push contact back
                if node_holder != nil {

                    k.GetContact_out <- node_holder.Value.(*Contact)

                } else {
                    k.GetContact_out <- nil
                }
        }
    }
}

func (k *Kademlia) ReturnLocalBuckets(req FindNodeRequest) []*Contact{
    //first get everything in the bucket the node requested would have gone in and put it in a FoundNode slice
    bucket_num := req.NodeID.Xor(k.Contact_table.NodeID).PrefixLen()
    bucket_slice := make([]*Contact,60)

    //counter for how many contacts we have stored
    counter := 0

    //counter for how much plus/minus we go from our original bucket
    bucketcounter := 0

    bucket := k.Contact_table.Buckets[bucket_num]
    for i := bucket.Front(); i != nil; i = i.Next() {
        bucket_slice[counter] = i.Value.(*Contact)
        counter++
    }
       
    //Check for out of bounds
    //this is a boolean, loop will run as long as it's false
    flag := bucket_num - bucketcounter < 0 && bucket_num + bucketcounter > 159

    //Then get everything from the bucket above and below unless you have 20 nodes already
    for len(bucket_slice) < ListSize && flag == false{

        //Get nodes from Buckets to the left
        if bucket_num - bucketcounter >= 0 {
            bucket = k.Contact_table.Buckets[bucket_num - bucketcounter]

            for i := bucket.Front(); i != nil; i = i.Next() {
                bucket_slice[counter] = i.Value.(*Contact)
                counter++
            }
        }

        //Get nodes from Buckets to the right
        if bucket_num + bucketcounter <= 159 {
            bucket = k.Contact_table.Buckets[bucket_num + bucketcounter]

            for i := bucket.Front(); i != nil; i = i.Next() {
                bucket_slice[counter] = i.Value.(*Contact)
                counter++
            }
        }

        //Increment bucket location counter
        bucketcounter++

        //update flag
        flag = bucket_num - bucketcounter < 0 && bucket_num + bucketcounter > 159
    }

    return bucket_slice
}

//error handling done, should be good 
func (k *Kademlia) DoPing(rhost net.IP, port uint16) error {

    //set up tcp
    address := rhost.String() +":"+ strconv.Itoa(int(port))
    client, err := rpc.DialHTTP("tcp", address)
    if err != nil {
        log.Printf("DialHTTP: ", err)
        return err
    }

    //fill out ping, initialize pong
    ping := new(Ping)
    ping.MsgID = NewRandomID()
    ping.Sender = *k.Info
    var pong Pong

    err = client.Call("Kademlia.Ping", ping, &pong)
    if err != nil {
        log.Printf("Call: ", err)
        return err
    }

    fmt.Printf(pong.Sender.Host.String())

    //run update with contact from pong struct
    err = k.Update(&pong.Sender)
    if err != nil {
        log.Printf("Update error: ", err)
        return err
    }

    //print confirmation
    fmt.Println("ping msgID: "+ ping.MsgID.AsString())
    fmt.Println("pong msgID: "+ pong.MsgID.AsString())

    //error handling and update
    if ping.MsgID.Compare(pong.MsgID) != 0 {
        err = errors.New("Ping and pong don't match.\n")
        log.Printf("Ping error: ", err)
        return err
    } else{
        //run update with contact from pong struct
        err = k.Update(&pong.Sender)
        if err != nil {
            log.Printf("Update error: ", err)
            return err
        }
    }

    return nil
}


func (k *Kademlia) DoStore(remoteContact *Contact, StoredKey ID, StoredValue []byte) error {

    //establish tcp
	address := remoteContact.Host.String() +":"+ strconv.Itoa(int(remoteContact.Port))
	client, err := rpc.DialHTTP("tcp", address)
    if err != nil {
        log.Printf("DialHTTP: ", err)
        return err
    }

    //fill out request
    req := new(StoreRequest)
    req.MsgID = NewRandomID()
    req.Sender = *k.Info
    req.Key = StoredKey
    req.Value = StoredValue
    var res StoreResult

    //call rpc
    err = client.Call("Kademlia.Store", req, &res)
    if err != nil {
        log.Printf("Call: ", err)
        return err
    }

    //print confirmation
    log.Printf("req msgID: %s\n", req.MsgID.AsString())
    log.Printf("res msgID: %s\n", res.MsgID.AsString())

    //error handling and update
    if req.MsgID.Compare(res.MsgID) != 0 {
        err = errors.New("Message IDs don't match.\n")
        log.Printf("Store error: ", err)
        return err
    } else{
        //run update with contact from pong struct
        err = k.Update(remoteContact)
        if err != nil {
            log.Printf("Update error: ", err)
            return err
        }
    }

    return nil
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
    fmt.Println(address)
	client, err := rpc.DialHTTP("tcp", address)
    if err != nil {
        log.Printf("DialHTTP: ", err)
        return nil, err
    }

    //make rpc call
    err = client.Call("Kademlia.FindNode", req, &res)
    if err != nil {
        log.Printf("Call: ", err)
        return nil, err
    }

    //error handling and update
    if req.MsgID.Compare(res.MsgID) != 0 {
        err = errors.New("Message IDs don't match.\n")
        log.Printf("FindNode error: ", err)
        return nil, err
    } else{
        //run update with contact from pong struct
        err = k.Update(remoteContact)
        if err != nil {
            log.Printf("Update error: ", err)
            return nil, err
        }
    }

    return res.Nodes, nil
}


//Needs testing
func (k *Kademlia) DoFindValue(remoteContact *Contact, Key ID) ([]byte, []FoundNode, error) {
	address := remoteContact.Host.String() + ":" + strconv.Itoa(int(remoteContact.Port))

	client, err := rpc.DialHTTP("tcp",address)
	if err != nil {
		log.Printf("DialHTTP:", err)
        return nil, nil, err
	}

	//request
	req := new(FindValueRequest)
	req.MsgID = NewRandomID()
	req.Sender = *k.Info
	req.Key = Key

	var res FindValueResult
	err = client.Call("Kademlia.FindValue", req, &res)
	if err != nil {
		log.Printf("Call: ", err)
        return nil, nil, err
	}

	k.Update(remoteContact)

	if res.Value == nil {
		fmt.Printf("No matching value!")
	} else {
		fmt.Printf("Match found!")
	}
	//Is there a way to output one or the other datatype?  
	//Not sure so just outputting both with a message to the user
	return res.Value, res.Nodes, nil
}


func (k *Kademlia) GetContact(searchid ID) *Contact {

    //push to BucketAccess
    k.GetContact_in <- searchid

    //pull back
    node_holder := <- k.GetContact_out
	if node_holder == nil{
		fmt.Printf("ERR\n")
        return nil
	}	else{
        return node_holder
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




func (k *Kademlia) IterativeFindNode(searchKey ID) ([]FoundNode, error) {
    
    // updated 20-element list containing ranked closest nodes
    short_list := make([]FoundNode,20)

    // // temp list to store findnode returned lists
    // temp_list := make([]FoundNode,20)

    //hashmap to check if nodes have been searched yet
    checkedMap := make(map[ID]int)


    // execute FindNode rpc to initialize the short_list from our own k buckets
    req := new(FindNodeRequest)
    req.MsgID = NewRandomID()
    req.Sender = *k.Info
    req.NodeID = searchKey
    var res FindNodeResult

    err := k.FindNode(*req, &res)
    short_list = res.Nodes

    if err != nil {
        log.Printf("IterativeFindNode could not make initial call: ", err)
        return nil, err
    }

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
            case temp_list := <-rcv_thread:
                // combine lists, remove duplicates, sort, trim to 20 elements
                new_list := make([]FoundNode,40)
                new_list = append(short_list, temp_list...)
                new_list = removeDuplicates(new_list)
                sort.Sort(ByDistanceFN(new_list))
                short_list = new_list[0:20]

                // check if closestNode is the same. If not, update 
                if short_list[0].NodeID.Compare(closestNode.NodeID) != 0 {
                    closestNode = short_list[0]
                    close_node_flag = 1
                }

                //increment return counter
                ret_counter++

            case err_node := <- err_ch:
                //remove from shortlist
                for j:=0; j<len(short_list); j++ {
                    if short_list[j].NodeID.Compare(err_node.NodeID) == 0 {
                        short_list = append(short_list[:j], short_list[j+1:]...)
                    }
                    break
                }
                //increment return counter
                ret_counter++
        }

        //enters if every cycle, check closest node not changing condition
        if ret_counter%Alpha == 0 && ret_counter!=0 {
            if close_node_flag == 0 {
                //ping everything in shortlist

                // done_ping_flag set to 1 when the entire short_list has been pinged
                done_ping_flag := 0
                j := 0

                for done_ping_flag == 0 {
                    if checkedMap[short_list[j].NodeID] == 0 {
                        //ping it
                        hostconverted, err := net.LookupIP(short_list[j].IPAddr)
                        err = k.DoPing(hostconverted[1], short_list[j].Port)
                        if err != nil{
                            //remove node from list
                            //WILL THIS WORK? Basically copying everything in slice except failed contact.
                            //will loop run correctly with re-indexing?
                            short_list = append(short_list[:j], short_list[j+1:]...)
                        } else {
                            // if successfully check with no error, move on
                            j++
                        }
                    } else {
                        // if that node has already been checked, move on
                        j++
                    }
                    if j == len(short_list) {
                        done_ping_flag = 1
                    }               
                }

                /*
                for j:=0; j<len(short_list); j++ {
                    if checkedMap[short_list[j].NodeID] == 0 {
                        //ping it
                        hostconverted, err := net.LookupIP(short_list[j].IPAddr)
                        err = k.DoPing(hostconverted[1], short_list[j].Port)
                        if err != nil{
                            //remove node from list
                            //WILL THIS WORK? Basically copying everything in slice except failed contact.
                            //will loop run correctly with re-indexing?
                            short_list = append(short_list[:j], short_list[j+1:]...)
                            j--
                        } 
                    }                    
                }
                */
                //end function
                loopFlag = false
            } else { 
                //reset flag
                close_node_flag = 0
            }
        }
    }

    return short_list, nil
}

func (k *Kademlia) FindNodeHandler(node *Contact, searchKey ID, rcv_thread chan []FoundNode, err_ch chan *Contact) {
    //return results of findnode through channel
    ret_list := make([]FoundNode,20)
    ret_list, err := k.DoFindNode(node, searchKey)

    if(err!=nil){
        err_ch <- node
    }else{        
        rcv_thread <- ret_list
    }
}

func (k *Kademlia) FindValueHandler(node *Contact, searchKey ID, rcv_value chan []byte, node_with_value chan *Contact, rcv_thread chan []FoundNode, err_ch chan *Contact) {
    //return results of findnode through channel
    ret_list := make([]FoundNode,20)
    found_value, ret_list, err := k.DoFindValue(node, searchKey)

    if err!=nil {
        err_ch <- node
    } else if found_value != nil {
        rcv_value <- found_value
        node_with_value <- node
    } else{        
        rcv_thread <- ret_list
    }
}

func (k *Kademlia) IterativeStore(key ID, value []byte) error {
    // call iterativeFindNode
    found_node_list, err := k.IterativeFindNode(key)

    if err != nil {
        return err
    }

    // iterate through foundnodelist, convert to contact pointers and call doStore
    for j:= 0; j<len(found_node_list); j++ {
        contact_ptr := new(Contact)
        hostconverted, err := net.LookupIP(found_node_list[j].IPAddr)
        contact_ptr.Host = hostconverted[1]
        contact_ptr.NodeID = found_node_list[j].NodeID
        contact_ptr.Port = found_node_list[j].Port

        // call doStore on converted contact pointer
        err = k.DoStore(contact_ptr, key, value)
        if err != nil {
            return err
        }
    }

    return nil
}


func (k *Kademlia) iterativeFindValue(searchKey ID) ([]byte, *Contact, []FoundNode, error) {

    // updated 20-element list containing ranked closest nodes
    short_list := make([]FoundNode,20)

    // // temp list to store findnode returned lists
    // temp_list := make([]FoundNode,20)

    //hashmap to check if nodes have been searched yet
    checkedMap := make(map[ID]int)


    // execute FindNode rpc to initialize the short_list from our own k buckets
    req := new(FindValueRequest)
    req.MsgID = NewRandomID()
    req.Sender = *k.Info
    //req.NodeID = searchKey
    var res FindValueResult

    err := k.FindValue(*req, &res)
    if res.Value != nil {
        return res.Value, k.Info, nil, nil
    }

    short_list = res.Nodes

    if err != nil {
        log.Printf("IterativeFindValuecould not make initial call: ", err)
        return nil, nil, nil, err
    }

    // set ClosestNode to the first element of short_list
    closestNode := short_list[0]

    //changed closest node flag
    var close_node_flag int = 0
    
    //these channels are each fed into FindNodeHandler
    //make rcv_thread channel
    rcv_thread := make(chan []FoundNode)
    //make rcv_value channel
    rcv_value := make(chan []byte)
    //make error rcv channel
    node_with_value := make(chan *Contact)
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
                        go k.FindValueHandler(&nodeToSearch, searchKey, rcv_value, node_with_value, rcv_thread, err_ch)   
                        loopFlag = false                
                    }
                }
                time.Sleep(300 * time.Millisecond)

            //case value returns
            case val := <-rcv_value:
                theonenode:= <- node_with_value
                return val, theonenode, nil, nil


            //thread returns successfully
            case temp_list := <-rcv_thread:
                // combine lists, remove duplicates, sort, trim to 20 elements
                new_list := make([]FoundNode,40)
                new_list = append(short_list, temp_list...)
                new_list = removeDuplicates(new_list)
                sort.Sort(ByDistanceFN(new_list))
                short_list = new_list[0:20]

                // check if closestNode is the same. If not, update 
                if short_list[0].NodeID.Compare(closestNode.NodeID) != 0 {
                    closestNode = short_list[0]
                    close_node_flag = 1
                }

                //increment return counter
                ret_counter++

            case err_node := <- err_ch:
                //remove from shortlist
                for j:=0; j<len(short_list); j++ {
                    if short_list[j].NodeID.Compare(err_node.NodeID) == 0 {
                        short_list = append(short_list[:j], short_list[j+1:]...)
                    }
                    break
                }
                //increment return counter
                ret_counter++
        }

        //enters if every cycle, check closest node not changing condition
        if ret_counter%Alpha == 0 && ret_counter!=0 {
            if close_node_flag == 0 {
                //ping everything in shortlist

                // done_ping_flag set to 1 when the entire short_list has been pinged
                done_ping_flag := 0
                j := 0

                for done_ping_flag == 0 {
                    if checkedMap[short_list[j].NodeID] == 0 {
                        //ping it
                        hostconverted, err := net.LookupIP(short_list[j].IPAddr)
                        err = k.DoPing(hostconverted[1], short_list[j].Port)
                        if err != nil{
                            //remove node from list
                            //WILL THIS WORK? Basically copying everything in slice except failed contact.
                            //will loop run correctly with re-indexing?
                            short_list = append(short_list[:j], short_list[j+1:]...)
                        } else {
                            // if successfully check with no error, move on
                            j++
                        }
                    } else {
                        // if that node has already been checked, move on
                        j++
                    }
                    if j == len(short_list) {
                        done_ping_flag = 1
                    }               
                }

              
                //end function
                loopFlag = false
            } else { 
                //reset flag
                close_node_flag = 0
            }
        }
    }
    
    return nil, nil, short_list, nil
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