// Routing functions for Whanau

package whanau

import "math/rand"

// Random walk
func (ws *WhanauServer) RandomWalk(args *RandomWalkArgs, reply *RandomWalkReply) error {
	var randomWalkReply RandomWalkReply
	if ws.is_sybil {
		randomWalkReply = ws.SybilRandomWalk()
		reply.Server = randomWalkReply.Server
		reply.Err = randomWalkReply.Err
	} else {
		//fmt.Printf("Doing an honest random walk\n")
		//randomWalkReply = ws.HonestRandomWalk(steps)
		nextServer, ok := ws.GetNextRWServer()
		if !ok {
			// Ran out of servers!!
			// Just go ahead and do a regular random walk
			steps := args.Steps
			randomWalkReply = ws.HonestRandomWalk(steps)
			reply.Server = randomWalkReply.Server
			reply.Err = randomWalkReply.Err
		} else {
			reply.Server = nextServer
			reply.Err = OK
		}
	}
	//fmt.Printf("Random walk reply: %s", randomWalkReply)
	return nil
}

// Random walk for honest nodes
func (ws *WhanauServer) HonestRandomWalk(steps int) RandomWalkReply {
	//fmt.Printf("In honest node random walk: %s", ws.myaddr)
	var reply RandomWalkReply
	// pick a random neighbor
	randIndex := rand.Intn(len(ws.neighbors))
	neighbor := ws.neighbors[randIndex]
	if steps == 1 {
		reply.Server = neighbor
		reply.Err = OK
	} else {
		args := RandomWalkArgs{}
		args.Steps = steps - 1
		var rpc_reply RandomWalkReply
		ok := call(neighbor, "WhanauServer.RandomWalk", args, &rpc_reply)
		if ok && (rpc_reply.Err == OK) {
			reply.Server = rpc_reply.Server
			reply.Err = OK
		} else {
			reply.Err = ErrNoKey
		}
	}
	return reply
}

// Random walk for sybil nodes
func (ws *WhanauServer) SybilRandomWalk() RandomWalkReply {
	//fmt.Printf("In Sybil node random walk: %s", ws.myaddr)
	// testing assumption for breaking cluster attacks

	if len(ws.neighbors) > 0 {
		randIndex := rand.Intn(len(ws.neighbors))
		neighbor := ws.neighbors[randIndex]
		return RandomWalkReply{neighbor, OK}
	} else {
		return RandomWalkReply{"Sybil server!", ErrNoKey}
	}
}

// Gets the ID from node's local id table
func (ws *WhanauServer) GetId(args *GetIdArgs, reply *GetIdReply) error {
	layer := args.Layer
	//DPrintf("In getid, len(ws.ids): %d layer: %d", len(ws.ids), layer)
	// gets the id associated with a layer
	if 0 <= layer && layer < len(ws.ids) {
		id := ws.ids[layer]
		reply.Key = id
		reply.Err = OK
	}
	return nil
}
