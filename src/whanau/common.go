package whanau

const (
	OK          = "OK"
	ErrNoKey    = "ErrNoKey"
	ErrRandWalk = "ErrRandWalk"
)

// for 2PC
const (
	// Action
	Commit = "commit"
	Abort  = "abort"

	// Reply
	Accept = "accept"
	Reject = "reject"

	// Phase
	PhaseOne = "p1"
	PhaseTwo = "p2"
)

type Err string

type KeyType string

type ValueType struct {
	Servers []string
}

type TrueValueType string

// Key value pair
type Record struct {
	Key   KeyType
	Value ValueType
}

// tuple for (id, address) pairs used in finger table
type Finger struct {
	Id      KeyType
	Address string
}

// Global Parameters
const (
	// n = number of honest nodes
	// m = O(n) number of honest edges
	// k = number of keys/node

	//W              = 2 // mixing time of honest region
	//RD             = 5 // size of db sqrt(km)
	//RF             = 5 // size of fingertable (per layer)
	//RS             = 5 // size of succ records
	//L              = 3 // number of layers log(km)
	PaxosSize = 5
	PaxosWalk = 5
	//STEPS          = 2 // number of steps to take in random walk
	//NUM_SUCCESSORS = 2 //number of successors obtained each step
	TIMEOUT = 5 // number of times to try querying
)
