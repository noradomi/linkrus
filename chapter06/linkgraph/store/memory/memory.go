package memory

import (
	"github.com/google/uuid"
	"golang.org/x/xerrors"
	_ "golang.org/x/xerrors"
	"linksrus/chapter06/linkgraph/graph"
	"sync"
	"time"
)

// Compile-time check for ensuring InMemoryGraph implements Graph.
//var _ graph.Graph = (*InMemoryGraph)(nil)

// edgeList contains the slice of edge UUIDs that originate from a link in the graph.
type edgeList []uuid.UUID

// InMemoryGraph implements an in-memory link graph that can be concurrently
// accessed by multiple clients.
type InMemoryGraph struct {
	mu sync.RWMutex

	links map[uuid.UUID]*graph.Link
	edges map[uuid.UUID]*graph.Edge

	linkURLIndex map[string]*graph.Link
	linkEdgeMap  map[uuid.UUID]edgeList // ? not *
}

// NewInMemoryGraph creates a new in-memory link graph.
func NewInMemoryGraph() *InMemoryGraph {
	return &InMemoryGraph{
		links:        make(map[uuid.UUID]*graph.Link),
		edges:        make(map[uuid.UUID]*graph.Edge),
		linkURLIndex: make(map[string]*graph.Link),
		linkEdgeMap:  make(map[uuid.UUID]edgeList),
	}
}

func (s *InMemoryGraph) UpsertLink(link *graph.Link) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing := s.linkURLIndex[link.URL]; existing != nil {
		link.ID = existing.ID
		origTs := existing.RetrievedAt
		*existing = *link
		if origTs.After(existing.RetrievedAt) {
			existing.RetrievedAt = origTs
		}
		return nil
	}

	// Loop to get unique uuid
	for {
		link.ID = uuid.New()
		if s.links[link.ID] == nil {
			break
		}
	}

	lCopy := new(graph.Link)
	*lCopy = *link
	s.linkURLIndex[lCopy.URL] = lCopy
	s.links[lCopy.ID] = lCopy
	return nil
}

func (s *InMemoryGraph) FindLink(id uuid.UUID) (*graph.Link, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	link := s.links[id]
	if link == nil {
		return nil, xerrors.Errorf("find link: %w", graph.ErrNotFound)
	}

	lCopy := new(graph.Link)
	*lCopy = *link
	return lCopy, nil
}

// ?: why not *
func (s *InMemoryGraph) Links(fromID, toID uuid.UUID, retrievedBefore time.Time) (graph.LinkIterator, error) {
	from, to := fromID.String(), toID.String()

	s.mu.Lock()
	var list []*graph.Link
	for linkID, link := range s.links {
		if id := linkID.String(); id >= from && id < to && link.RetrievedAt.Before(retrievedBefore) {
			list = append(list, link)
		}
	}
	s.mu.Unlock()

	return &linkIterator{s: s, links: list}, nil
}

func (s *InMemoryGraph) UpsertEdge(edge *graph.Edge) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, srcExists := s.links[edge.Src]
	_, dstExists := s.links[edge.Dst]

	if !srcExists || !dstExists {
		return xerrors.Errorf("upsert edge: %w", graph.ErrUnknownEdgeLinks)
	}
	for _, edgeID := range s.linkEdgeMap[edge.Src] {
		existingEdge := s.edges[edgeID]
		if existingEdge.Src == edge.Src && existingEdge.Dst == edge.Dst {
			existingEdge.UpdatedAt = time.Now()
			*edge = *existingEdge
			return nil
		}
	}

	return nil
}
