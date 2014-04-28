// Whanau server serves keys when it can.

package whanau

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"sort"
	"sync"
)

//import "builtin"

//import "encoding/gob"

const Debug = 1

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug > 0 {
		log.Printf(format, a...)
	}
	return
}

type WhanauServer struct {
	mu     sync.Mutex
	l      net.Listener
	me     int
	myaddr string
	dead   bool // for testing

	neighbors []string                  // list of servers this server can talk to
	pkvstore  map[KeyType]TrueValueType // local k/v table, used for Paxos
	kvstore   map[KeyType]ValueType     // k/v table used for routing
	ids       []KeyType                 // contains id of each layer
	fingers   [][]Finger                // (id, server name) pairs
	succ      [][]Record                // contains successor records for each layer
	db        []Record                  // sample of records used for constructing struct, according to the paper, the union of all dbs in all nodes cover all the keys =)
}

func IsInList(val string, array []string) bool {
	for _, v := range array {
		if v == val {
			return true
		}
	}

	return false
}

func (ws *WhanauServer) PaxosLookup(servers ValueType) TrueValueType {
	// TODO: make a call to a random server in the group
	return ""
}

// TODO this eventually needs to become a real lookup
func (ws *WhanauServer) Lookup(args *LookupArgs, reply *LookupReply) error {
	if val, ok := ws.kvstore[args.Key]; ok {
		var ret TrueValueType
		ret = ws.PaxosLookup(val)

		reply.Value = ret
		reply.Err = OK
		return nil
	}

	// probe neighbors
	// TODO eventually needs to look up based on successor table
	routedFrom := args.RoutedFrom
	routedFrom = append(routedFrom, ws.myaddr)
	neighborVal := ws.NeighborLookup(args.Key, routedFrom)

	// TODO this is a hack. NeighborLookup should be changed
	// to actually return an error.
	if neighborVal != ErrNoKey {
		reply.Value = neighborVal
		reply.Err = OK
		return nil
	}

	reply.Err = ErrNoKey
	return nil
}

// Client-style lookup on neighboring servers.
// routedFrom is supposed to prevent infinite lookup loops.
func (ws *WhanauServer) NeighborLookup(key KeyType, routedFrom []string) TrueValueType {
	args := &LookupArgs{}
	args.Key = key
	args.RoutedFrom = routedFrom
	var reply LookupReply
	for _, srv := range ws.neighbors {
		if IsInList(srv, routedFrom) {
			continue
		}

		ok := call(srv, "WhanauServer.Lookup", args, &reply)
		if ok && (reply.Err == OK) {
			return reply.Value
		}
	}

	return ErrNoKey
}

func (ws *WhanauServer) PaxosPut(key KeyType, value TrueValueType) error {
	// TODO: needs to do a real paxos put
	ws.pkvstore[key] = value
	return nil
}

// TODO this eventually needs to become a real put
func (ws *WhanauServer) Put(args *PutArgs, reply *PutReply) error {
	// TODO: needs to 1. find the paxos cluster 2. do a paxos cluster put
	reply.Err = OK
	return nil
}

// Random walk
func (ws *WhanauServer) RandomWalk(args *RandomWalkArgs, reply *RandomWalkReply) error {
	steps := args.Steps
	// pick a random neighbor
	randIndex := rand.Intn(len(ws.neighbors))
	neighbor := ws.neighbors[randIndex]
	if steps == 1 {
		reply.Server = neighbor
		reply.Err = OK
	} else {
		args := &RandomWalkArgs{}
		args.Steps = steps - 1
		var rpc_reply RandomWalkReply
		ok := call(neighbor, "WhanauServer.RandomWalk", args, &rpc_reply)
		if ok && (rpc_reply.Err == OK) {
			reply.Server = rpc_reply.Server
			reply.Err = OK
		}
	}

	return nil
}

// Gets the ID from node's local id table
func (ws *WhanauServer) GetId(args *GetIdArgs, reply *GetIdReply) error {
	layer := args.Layer
	DPrintf("In getid, len(ws.ids): %d", len(ws.ids))
	// gets the id associated with a layer
	if 0 <= layer && layer <= len(ws.ids) {
		id := ws.ids[layer]
		DPrintf("In getid rpc id: %s", id)
		reply.Key = id
		reply.Err = OK
	}
	return nil
}

// Whanau Routing Protocal methods

// TODO
// Populates routing table
func (ws *WhanauServer) Setup(nlayers int, rf int) {
	DPrintf("In Setup of server %s", ws.myaddr)
	// fill up db by randomly sampling records from random walks
	// "The db table has the good property that each honest node’s stored records are frequently represented in other honest nodes’db tables"
	ws.db = ws.SampleRecords(RD)
	// reset ids, fingers
	ws.ids = make([]KeyType, 0)
	ws.fingers = make([][]Finger, 0)
	// TODO add successors
	for i := 0; i < nlayers; i++ {
		// populate tables in layers
		ws.ids = append(ws.ids, ws.ChooseID(i))
		ws.fingers = append(ws.fingers, ws.ConstructFingers(i, rf))
		// TODO add sucessors
	}

}

// return random Key/value record from local storage
func (ws *WhanauServer) SampleRecord() Record {
	randIndex := rand.Intn(len(ws.kvstore))
	keys := make([]KeyType, 0)
	for k, _ := range ws.kvstore {
		keys = append(keys, k)
	}
	key := keys[randIndex]
	value := ws.kvstore[key]
	record := Record{key, value}

	return record
}

// Returns a list of records sampled randomly from local kv store
// Note: we agreed that duplicates are fine
func (ws *WhanauServer) SampleRecords(rd int) []Record {

	records := make([]Record, 0)
	for i := 0; i < rd; i++ {
		records = append(records, ws.SampleRecord())
	}
	return records
}

// Constructs Finger table for a specified layer
func (ws *WhanauServer) ConstructFingers(layer int, rf int) []Finger {
	fingers := make([]Finger, 0)
	for i := 0; i < rf; i++ {
		steps := W // TODO: set to global W parameter
		args := &RandomWalkArgs{steps}
		reply := &RandomWalkReply{}

		// Keep trying until succeed or timeout
		// TODO add timeout later
		for reply.Err != OK {
			DPrintf("random walk")
			ws.RandomWalk(args, reply)
		}
		server := reply.Server

		DPrintf("randserver: %s", server)
		// get id of server using rpc call to that server
		getIdArg := &GetIdArgs{layer}
		getIdReply := &GetIdReply{}
		ok := false

		// block until succeeds
		// TODO add timeout later
		for !ok || (getIdReply.Err != OK) {
			DPrintf("rpc to getid")
			ok = call(server, "WhanauServer.GetId", getIdArg, getIdReply)
		}

		finger := Finger{getIdReply.Key, server}
		fingers = append(fingers, finger)
	}

	return fingers
}

// Choose id for specified layer
func (ws *WhanauServer) ChooseID(layer int) KeyType {

	if layer == 0 {
		DPrintf("In ChooseID, layer 0")
		// choose randomly from db
		randIndex := rand.Intn(len(ws.db))
		record := ws.db[randIndex]
		DPrintf("record.Key", record.Key)
		return record.Key

	} else {
		// choose finger randomly from layer - 1, use id of that finger
		randFinger := ws.fingers[layer-1][rand.Intn(len(ws.fingers[layer-1]))]
		return randFinger.Id
	}
}

// Defines ordering of Record args
type By func(p1, p2 *Record) bool

// Sort uses By to sort the Record slice
func (by By) Sort(records []Record) {
	rs := &recordSorter{
		records: records,
		by:      by,
	}
	sort.Sort(rs)
}

// recordSorter joins a By function and a slice of Records to be sorted.
type recordSorter struct {
	records []Record
	by      func(p1, p2 *Record) bool // Closure used in the Less method.
}

// Len is part of sort.Interface.
func (s *recordSorter) Len() int {
	return len(s.records)
}

// Swap is part of sort.Interface.
func (s *recordSorter) Swap(i, j int) {
	s.records[i], s.records[j] = s.records[j], s.records[i]
}

// Less is part of sort.Interface. It is implemented by calling the "by" closure in the sorter.
func (s *recordSorter) Less(i, j int) bool {
	return s.by(&s.records[i], &s.records[j])
}

// Gets successors that are nearest each key
func (ws *WhanauServer) SampleSuccessors(args *SampleSuccessorsArgs, reply *SampleSuccessorsReply) error {
	recordKey := func(r1, r2 *Record) bool {
		return r1.Key < r2.Key
	}
	By(recordKey).Sort(ws.db)

	key := args.Key
	t := args.T
	var records []Record
	curCount := 0
	curRecord := 0
	if t <= len(ws.db) {
		for curCount < t {
			if ws.db[curRecord].Key >= key {
				records = append(records, ws.db[curRecord])
				curCount++
			}
			curRecord++
			if curRecord == len(ws.db) {
				curRecord = 0
				key = ws.db[curRecord].Key
			}
		}
		reply.Successors = records
		reply.Err = OK
	} else {
		reply.Err = ErrNoKey
	}
	return nil
}

func (ws *WhanauServer) Successors(layer int) []Record {
	var successors []Record
	for i := 0; i < RS; i++ {
		args := &RandomWalkArgs{}
		args.Steps = STEPS
		reply := &RandomWalkReply{}
		ws.RandomWalk(args, reply)

		if reply.Err == OK {
			vj := reply.Server
			getIdArgs := &GetIdArgs{layer}
			getIdReply := &GetIdReply{}
			ws.GetId(getIdArgs, getIdReply)

			sampleSuccessorsArgs := &SampleSuccessorsArgs{getIdReply.Key, NUM_SUCCESSORS}
			sampleSuccessorsReply := &SampleSuccessorsReply{}
			for sampleSuccessorsReply.Err != OK {
				call(vj, "Whanau.SampleSuccessors", sampleSuccessorsArgs, sampleSuccessorsReply)
			}
			successors = append(successors, sampleSuccessorsReply.Successors...)
		}
	}
	return successors
}

// tell the server to shut itself down.
func (ws *WhanauServer) kill() {
	ws.dead = true
	ws.l.Close()
	//	ws.px.Kill()
}

// TODO servers is for a paxos cluster
func StartServer(servers []string, me int, myaddr string, neighbors []string) *WhanauServer {
	ws := new(WhanauServer)
	ws.me = me
	ws.myaddr = myaddr
	ws.neighbors = neighbors

	ws.kvstore = make(map[KeyType]ValueType)
	ws.pkvstore = make(map[KeyType]TrueValueType)

	rpcs := rpc.NewServer()
	rpcs.Register(ws)

	os.Remove(servers[me])
	l, e := net.Listen("unix", servers[me])
	if e != nil {
		log.Fatal("listen error: ", e)
	}
	ws.l = l

	go func() {
		for ws.dead == false {
			conn, err := ws.l.Accept()
			// removed unreliable code for now
			if err == nil && ws.dead == false {
				go rpcs.ServeConn(conn)
			} else if err == nil {
				conn.Close()
			}

			if err != nil && ws.dead == false {
				fmt.Printf("ShardWS(%v) accept: %v\n", me, err.Error())
				ws.kill()
			}
		}
	}()

	// removed tick() loop for now

	return ws
}

// Methods used only for testing

// This method is only used for putting ids into the table for testing purposes
func (ws *WhanauServer) PutId(args *PutIdArgs, reply *PutIdReply) error {
	//ws.ids[args.Layer] = args.Key
  ws.ids = append(ws.ids, args.Key)
	reply.Err = OK
	return nil
}
