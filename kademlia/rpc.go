package kademlia
// Contains definitions mirroring the Kademlia spec. You will need to stick
// strictly to these to be compatible with the reference implementation and
// other groups' code.

import "net"
import "sort"
import "log"


// Host identification.
type Contact struct {
    NodeID ID
    Host net.IP
    Port uint16
    // add public keys
}

type private_key struct {
    key string
}

// PING
type Ping struct {
    Sender Contact
    MsgID ID
}

type Pong struct {
    MsgID ID
    Sender Contact
}

func (k *Kademlia) Ping(ping Ping, pong *Pong) error {
    // This one's a freebie.
    pong.MsgID = CopyID(ping.MsgID)
    pong.Sender = *k.Info
        log.Printf("ping msgIDrpc:\n")

    if err := k.Update(&ping.Sender); err != nil {
        return err
    } else {
        return nil
    }
}


// STORE
type StoreRequest struct {
    Sender Contact
    MsgID ID
    Key ID
    Value []byte
}

type StoreResult struct {
    MsgID ID
    Err error
}

func (k *Kademlia) Store(req StoreRequest, res *StoreResult) error {
    k.Store_key <- req.Key
    k.Store_val <- req.Value

    //k.Bin[req.Key] = req.Value
    res.MsgID = CopyID(req.MsgID)
    if err := k.Update(&req.Sender); err != nil {
        return err
    } else {
        return nil
    }
}


// FIND_NODE
type FindNodeRequest struct {
    Sender Contact
    MsgID ID
    NodeID ID
}

type FoundNode struct {
    IPAddr string
    Port uint16
    NodeID ID
    // public key
}

type FindNodeResult struct {
    MsgID ID
    Nodes []FoundNode
    Err error
}

type ByDistance []*Contact

func (a ByDistance) Swap(i,j int)       {a[i], a[j] = a[j], a[i]}
func (a ByDistance) Len() int           {return len(a)}
func (a ByDistance) Less(i, j int) bool {return a[i].NodeID.Less(a[j].NodeID)}


// WE NEED TO EDIT FINDNODE TO ADD PUBLIC KEYS TO FOUNDNODE RESULT ARRAY 
func (k *Kademlia) FindNode(req FindNodeRequest, res *FindNodeResult) error {
    
    //push req to BucketAccess
    k.FindNode_in_ch <- req

    //pull bucketslice from BucketAccess
    bucket_slice := <- k.FindNode_out_ch

    //Sort the slice by Xor distance to the input nodeID
    sort.Sort(ByDistance(bucket_slice))

    //Get 20 closest
    bucket_slice = bucket_slice[0:20]
    FoundNodes := make([]FoundNode,20)

    for i := 0; i<20; i++ {
        FoundNodes[i].IPAddr = bucket_slice[i].Host.String()
        FoundNodes[i].Port = bucket_slice[i].Port
        FoundNodes[i].NodeID = bucket_slice[i].NodeID
    }

    //Fill out result struct
    res.Nodes = FoundNodes
    res.MsgID = CopyID(req.MsgID)


    if err := k.Update(&req.Sender); err != nil {
        return err
    } else {
        return nil
    }
}


// FIND_VALUE
type FindValueRequest struct {
    Sender Contact
    MsgID ID
    Key ID
}

// If Value is nil, it should be ignored, and Nodes means the same as in a
// FindNodeResult.
type FindValueResult struct {
    MsgID ID
    Value []byte
    Nodes []FoundNode
    Err error
}

//NEED TO PUT IN CONTACT_TABLE LOCKS
func (k *Kademlia) FindValue(req FindValueRequest, res *FindValueResult) error {
    k.FindVal_in <- req.Key
    map_value := <- k.FindVal_out
    //map_value := k.Bin[req.Key]
    if map_value != nil {
        res.Value = map_value
    } else {
        freq := new(FindNodeRequest)
        freq.MsgID = req.MsgID
        freq.Sender = req.Sender
        freq.NodeID = req.Key
        var fres = new(FindNodeResult)
        k.FindNode(*freq, fres)

        res.Nodes = fres.Nodes
        res.MsgID = CopyID(req.MsgID)
        res.Err = nil
        res.Value = nil
    }


    return nil
}

type ForwardRequest struct {
    Destination Contact
    RequestIDprev int
    RequestIDnext int
    Sender Contact
    HopCntr int // 0 = you are at the destination
    itemID ID // used to extract byte array from destination's map
}

type ForwardResponse struct {
    Payload []byte
    RequestID int
}

func (k *Kademlia) Forward_Handle (req ForwardRequest, res *ForwardResponse) error {
    
    // if back at Sender (and direction = -1) -> done
    
         
    if k.info.NodeID == req.Destination {
        res.Payload = k.Bin[req.itemID] // extract data from destination
        res.RequestID = req.RequestID
    }
    else {
        if req.HopCntr == 0 {
            // update forwarding table
            // DoF(req.Destination, 0, req.Destination, msgID, itemID)
        } else {
            // update forwarding table
            // temp = random node (that hasn't been picked)
            //      update forwarding table
            //      update HopCntr
            //      DoF(temp, cntr, destination, msgID, itemID)
        }
    }
    return error
}

    // if at Destination -> Reverse direction and call backwards
    // else -> call in the direction indicated 
    // 

}
