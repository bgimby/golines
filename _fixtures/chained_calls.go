package fixtures

type Chain struct{}

func (c *Chain) ChainCall(arg1 string, arg2 string, arg3 string) *Chain {
	return c
}

func NewChain() *Chain {
	return &Chain{}
}

func ChainedCalls() {
	c := Chain{}
	NewChain().
		ChainCall("a", "b", "c")
	c.
		ChainCall("a", "b", "c")
}
