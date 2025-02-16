package node

type Node struct {
	Name            string
	Ip              string
	Api             string
	Memory          int
	MemoryAllocated int
	Disk            int
	DiskAllocated   int
	Cores           int
	// Stats           stats.Stats
	Role      string
	TaskCount int
}

func NewNode(role string, nApi string, name string) *Node {
	return &Node{
		Name: name,
		Api:  nApi,
		Role: role,
	}
}
