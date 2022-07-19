package memory

import (
	gc "gopkg.in/check.v1"
	"linksrus/chapter06/linkgraph/graph/graphtest"
	"testing"
)

var _ = gc.Suite(new(InMemoryTestSuite))

func Test(t *testing.T) {
	gc.TestingT(t)
}

type InMemoryTestSuite struct {
	graphtest.SuiteBase
}

func (s *InMemoryTestSuite) SetUpTest(c *gc.C) {
	s.SetGraph(NewInMemoryGraph())
}
