package merkle

import "github.com/HPISTechnologies/component-lib/actor"

type Node struct {
	isStatic    bool
	sharedKey   *string // For nonleaf nodes will share the same path with their children
	childOffset uint8_t
	numChildren uint32_t
}

func (this *Node) MakeStatic() {
	this.isStatic = true
}

func NewNode(concurrency int, groupId string) actor.IWorkerEx {

}
