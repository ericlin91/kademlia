package kademlia
// Contains definitions mirroring the Kademlia spec. You will need to stick
// strictly to these to be compatible with the reference implementation and
// other groups' code.

import "net"
import "sort"


// Host identification.
type Contact struct {
    NodeID ID
    Host net.IP
    Port uint16
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
    pong.Sender = *k.info
    k.Update(&ping.Sender)
    return nil
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
    k.bin[req.Key] = req.Value
    k.Update(&req.Sender)
    res.MsgID = CopyID(req.MsgID)
    return nil
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

func (k *Kademlia) FindNode(req FindNodeRequest, res *FindNodeResult) error {
    //first get everything in the bucket the node requested would have gone in and put it in a FoundNode slice
    bucket_num := node.NodeID.Xor(k.table.NodeID).PrefixLen()
    bucket_slice := ([]*Contact)

    counter := 0
    bucketcounter := 0

    bucket := k.table.buckets[bucket_num]
    for i := bucket.Front(); i != nil; i = i.Next() {
        bucket_slice[counter] = i.Value.(*Contact)
        counter++
    }

    //Then get everything from the bucket above and below unless you have 20 nodes already
    for bucket_slice.Len() < 20 && flag == false{

        //Get nodes from buckets to the left
        if bucket_num - bucketcounter >= 0 {
            bucket = k.table.buckets[bucket_num - bucketcounter]

            for i := bucket.Front(); i != nil; i = i.Next() {
                bucket_slice[counter] = i.Value.(*Contact)
                counter++
            }
        }

        //Get nodes from buckets to the right
        if bucket_num + bucketcounter <= 159 {
            bucket = k.table.buckets[bucket_num + bucketcounter]

            for i := bucket.Front(); i != nil; i = i.Next() {
                bucket_slice[counter] = i.Value.(*Contact)
                counter++
            }
        }

        //Increment bucket location counter
        bucketcounter++

        //Check for out of bounds
        flag := bucket_num - bucketcounter < 0 && bucket_num + bucketcounter > 159
    }

    //Sort the slice by Xor distance to the input nodeID
    sort.Sort(ByDistance(bucket_slice))

    //Get 20 closest
    bucket_slice = bucket_slice[0:19]
    FoundNodes := ([]FoundNode)

    for i := 0 {
        FoundNodes[i].IPAddr = bucket_slice[i].Host.String()
        FoundNodes[i].Port = bucket_slice[i].Port
        FoundNodes[i].NodeID = bucket_slice[i].NodeID
    }

    //Fill out result struct
    res.Nodes = FoundNodes

    //Update not called.  Important?  IDK
    return nil
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

func (k *Kademlia) FindValue(req FindValueRequest, res *FindValueResult) error {
    // TODO: Implement.
    return nil
}

