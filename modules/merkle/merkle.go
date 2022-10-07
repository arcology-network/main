package merkle

import (
	"fmt"

	ethCommon "github.com/HPISTechnologies/3rd-party/eth/common"
	"github.com/HPISTechnologies/common-lib/types"
	"github.com/HPISTechnologies/main/modules/scheduler/lib"
)

type Merkle struct {
	Node<B, T, H>* root;
	Pool<char*>* keyPool;
	Pool<Node<B, T, H>*>* nodePool;
}
