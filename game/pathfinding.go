package game

import (
	"container/heap"

	"github.com/kivutar/goro/game/scene"
	"github.com/kivutar/goro/res"
	worldstate "github.com/kivutar/goro/world"
)

func walkPath(gat *res.GAT, fromX, fromY, toX, toY int) []worldstate.WalkStep {
	if fromX == toX && fromY == toY {
		return []worldstate.WalkStep{{X: fromX, Y: fromY}}
	}
	if gat == nil || !gat.InBounds(fromX, fromY) || !gat.InBounds(toX, toY) {
		return []worldstate.WalkStep{{X: fromX, Y: fromY}, {X: toX, Y: toY}}
	}
	if path, ok := findWalkPath(gat, fromX, fromY, toX, toY); ok {
		return path
	}
	return []worldstate.WalkStep{{X: fromX, Y: fromY}, {X: toX, Y: toY}}
}

func findWalkPath(gat *res.GAT, fromX, fromY, toX, toY int) ([]worldstate.WalkStep, bool) {
	start := pathPoint{x: fromX, y: fromY}
	goal := pathPoint{x: toX, y: toY}
	open := &pathHeap{}
	heap.Init(open)
	startNode := &pathNode{point: start, g: 0, f: pathHeuristic(start, goal)}
	heap.Push(open, startNode)
	nodes := map[pathPoint]*pathNode{start: startNode}
	closed := make(map[pathPoint]struct{})

	for open.Len() > 0 {
		current := heap.Pop(open).(*pathNode)
		if _, ok := closed[current.point]; ok {
			continue
		}
		if current.point == goal {
			return reconstructPath(current), true
		}
		closed[current.point] = struct{}{}
		for _, next := range pathNeighbors(gat, current.point) {
			if _, ok := closed[next]; ok {
				continue
			}
			cost := current.g + pathStepCost(current.point, next)
			node, ok := nodes[next]
			if !ok {
				node = &pathNode{point: next, g: cost, f: cost + pathHeuristic(next, goal), parent: current}
				nodes[next] = node
				heap.Push(open, node)
				continue
			}
			if cost < node.g {
				node.g = cost
				node.f = cost + pathHeuristic(next, goal)
				node.parent = current
				heap.Push(open, node)
			}
		}
	}
	return nil, false
}

type pathPoint struct {
	x int
	y int
}

type pathNode struct {
	point  pathPoint
	g      int
	f      int
	parent *pathNode
}

type pathHeap []*pathNode

func (h pathHeap) Len() int { return len(h) }

func (h pathHeap) Less(i, j int) bool {
	if h[i].f == h[j].f {
		return h[i].g > h[j].g
	}
	return h[i].f < h[j].f
}

func (h pathHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *pathHeap) Push(x any) { *h = append(*h, x.(*pathNode)) }

func (h *pathHeap) Pop() any {
	old := *h
	n := len(old)
	node := old[n-1]
	*h = old[:n-1]
	return node
}

func pathNeighbors(gat *res.GAT, point pathPoint) []pathPoint {
	var out []pathPoint
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			next := pathPoint{x: point.x + dx, y: point.y + dy}
			if !gat.InBounds(next.x, next.y) {
				continue
			}
			if !gat.Walkable(next.x, next.y) {
				continue
			}
			if dx != 0 && dy != 0 && (!gat.Walkable(point.x+dx, point.y) || !gat.Walkable(point.x, point.y+dy)) {
				continue
			}
			out = append(out, next)
		}
	}
	return out
}

func pathStepCost(from, to pathPoint) int {
	if from.x != to.x && from.y != to.y {
		return 14
	}
	return 10
}

func pathHeuristic(from, to pathPoint) int {
	dx := scene.AbsInt(to.x - from.x)
	dy := scene.AbsInt(to.y - from.y)
	if dx < dy {
		return 14*dx + 10*(dy-dx)
	}
	return 14*dy + 10*(dx-dy)
}

func reconstructPath(node *pathNode) []worldstate.WalkStep {
	var reversed []worldstate.WalkStep
	for node != nil {
		reversed = append(reversed, worldstate.WalkStep{X: node.point.x, Y: node.point.y})
		node = node.parent
	}
	path := make([]worldstate.WalkStep, len(reversed))
	for i := range reversed {
		path[i] = reversed[len(reversed)-1-i]
	}
	return path
}
