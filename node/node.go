package node

type Node struct {
	Name            string
	Ip              string
	Cores           int
	Memory          int
	MemoryAllocated int
	Disk            int
	DiskAllocated   int
	Role            string
	TaskCount       int
}

func NewNode(role string, nApi string, name string) *Node {
	newNode := Node{}

	newNode.Name = name
	newNode.Ip = nApi
	newNode.Role = role

	return &newNode
}
