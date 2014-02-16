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
    k.Bin[req.Key] = req.Value
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
    
    map_value := k.Bin[req.Key]
    if map_value != nil {
        res.Value = map_value
    } else {
        bucket_num := k.Info.NodeID.Xor(k.Contact_table.NodeID).PrefixLen()
        bucket_slice := make([]*Contact,60)

        counter := 0
        bucketcounter := 0

        searching_bucket := k.Contact_table.Buckets[bucket_num]
        for i := searching_bucket.Front(); i != nil; i = i.Next() {
            bucket_slice[counter] = i.Value.(*Contact)
            counter++
        }

        flag := bucket_num - bucketcounter < 0 && bucket_num + bucketcounter > 159

        for len(bucket_slice)< 20 && flag == false {
            if bucket_num - bucketcounter >= 0 {
                searching_bucket = k.Contact_table.Buckets[bucket_num - bucketcounter]

                for i := searching_bucket.Front(); i != nil; i = i.Next() {
                    bucket_slice[counter] = i.Value.(*Contact)
                    counter++
                }
            }

            if bucket_num + bucketcounter <= 159 {
                searching_bucket = k.Contact_table.Buckets[bucket_num + bucketcounter]

                for i := searching_bucket.Front(); i != nil; i = i.Next() {
                    bucket_slice[counter] = i.Value.(*Contact)
                    counter++
                }
            }

            bucketcounter++
            flag = bucket_num - bucketcounter < 0 && bucket_num + bucketcounter > 159
        }
        sort.Sort(ByDistance(bucket_slice))
        bucket_slice = bucket_slice[0:19]
        FoundNodes := make([]FoundNode,60)

        for i := 0; i<20; i++ {
            FoundNodes[i].IPAddr = bucket_slice[i].Host.String()
            FoundNodes[i].Port = bucket_slice[i].Port
            FoundNodes[i].NodeID = bucket_slice[i].NodeID
        }
        res.Nodes = FoundNodes
    }


    return nil
}


/*
old findnode

func (k *Kademlia) FindNode(req FindNodeRequest, res *FindNodeResult) error {
    
    //first get everything in the bucket the node requested would have gone in and put it in a FoundNode slice
    bucket_num := req.NodeID.Xor(k.Contact_table.NodeID).PrefixLen()
    bucket_slice := make([]*Contact,60)

    //counter for how many contacts we have stored
    counter := 0

    //counter for how much plus/minus we go from our original bucket
    bucketcounter := 0

    //get lock for contact_table
    k.Table_ch <- 1

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

    //release lock for contact_table
    <-k.Table_ch

    //Sort the slice by Xor distance to the input nodeID
    sort.Sort(ByDistance(bucket_slice))

    //Get 20 closest
    bucket_slice = bucket_slice[0:19]
    FoundNodes := make([]FoundNode,60)

    for i := 0; i<20; i++ {
        FoundNodes[i].IPAddr = bucket_slice[i].Host.String()
        FoundNodes[i].Port = bucket_slice[i].Port
        FoundNodes[i].NodeID = bucket_slice[i].NodeID
    }

    //Fill out result struct
    res.Nodes = FoundNodes

    if err := k.Update(&req.Sender); err != nil {
        return err
    } else {
        return nil
    }
}*/