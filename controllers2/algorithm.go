package controllers2

import (
	"fmt"
	"math"
	"sort"
	"time"

	dijkstrav2 "jinli.io/crdshortestpath/api/dijkstra/v2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ComputeShortestPath(kn *dijkstrav2.KnownNodes, dp *dijkstrav2.Display) {
	if !startIDInNodes(kn, dp) {
		var targetNodes []dijkstrav2.TargetNode
		dp.Status.TargetNodes = targetNodes
		dp.Status.LastUpdate = v1.NewTime(time.Now())
		dp.Status.ComputeStatus = "Failed"
	} else {
		if dp.Spec.Algorithm == "dijkstra" {
			var targetNodes []dijkstrav2.TargetNode
			var nodes []*dijkstrav2.Node
			for i := range kn.Spec.Nodes {
				nodes = append(nodes, &kn.Spec.Nodes[i])
			}
			// 计算起始节点到所有节点最短路径
			distance := dijkstra(nodes, dp.Spec.StartNode.ID)
			for id, dist := range distance {
				var path []string
				var name string
				for _, node := range nodes {
					if node.ID != id {
						continue
					}
					name = node.Name
					// 倒着追加
					preNode := node.PreNode
					for preNode != nil {
						path = append(path, preNode.Name)
						preNode = preNode.PreNode
					}
					break
				}
				// 反转输出
				reverse(path)
				var targetNode dijkstrav2.TargetNode
				targetNode.Distance = int32(dist)
				targetNode.ID = id
				targetNode.Name = name
				targetNode.Path = fmt.Sprintf("%s", path)
				targetNodes = append(targetNodes, targetNode)
			}
			dp.Status.TargetNodes = targetNodes
			dp.Status.LastUpdate = v1.NewTime(time.Now())
			dp.Status.ComputeStatus = "Succeed"
		}
	}
}

func dijkstra(nodes []*dijkstrav2.Node, startNode int32) map[int32]uint {
	// 记录节点是否被访问
	visited := make(map[int32]bool)
	// 记录起始节点到各个节点最短距离
	distance := make(map[int32]uint)
	// 记录节点的上一个节点
	preNode := make(map[int32]*dijkstrav2.Node)

	for _, node := range nodes {
		distance[node.ID] = math.MaxInt
		preNode[node.ID] = nil
	}

	// 起始节点到自身的距离为0，也即为第一个minnode
	distance[startNode] = 0
	// 循环到所有节点都被访问，或者找不到距离起始节点最近的未被访问的节点
	for len(visited) < len(nodes) {
		min := &dijkstrav2.Node{ID: -1, Name: "MinNode"}
		distance[int32(min.ID)] = math.MaxInt
		// 找到距离起始节点最近的未被访问的节点
		for _, node := range nodes {
			if !visited[node.ID] && distance[node.ID] < distance[int32(min.ID)] {
				min = node
			}
		}

		// 找不到未被访问的节点，或者找不到距离起始节点最近的未被访问的节点，跳出循环
		if min.ID == -1 {
			break
		} else {
			// 标记当前节点为已被访问
			visited[min.ID] = true
		}

		for _, edge := range min.Edges {
			// 更新min节点其相邻节点距离记录
			newDistance := distance[min.ID] + uint(edge.Cost)
			if newDistance < distance[edge.To] {
				distance[edge.To] = newDistance
				preNode[edge.To] = min
			}
		}
	}

	// 更新每个节点的父节点
	for _, node := range nodes {
		node.PreNode = preNode[node.ID]
	}
	// 去除初始节点距离记录
	delete(distance, -1)
	return distance
}

// 计算dp的startID是否在kn对象的nodes中
func startIDInNodes(kn *dijkstrav2.KnownNodes, dp *dijkstrav2.Display) bool {
	var flag int
	for _, node := range kn.Spec.Nodes {
		if dp.Spec.StartNode.ID != node.ID {
			flag++
			continue
		}
		break
	}

	return flag != len(kn.Spec.Nodes)
}

// 反转切片
func reverse(slice []string) {
	for i, j := 0, len(slice)-1; i < j; i, j = i+1, j-1 {
		slice[i], slice[j] = slice[j], slice[i]
	}
}

// 比较Nodes
func NodesEqual(s1, s2 []dijkstrav2.Node) bool {
	sort.Sort(NodesByID(s1))
	sort.Sort(NodesByID(s2))
	if len(s1) != len(s2) {
		return false
	}
	for i := range s1 {
		if s1[i].ID != s2[i].ID || s1[i].Name != s2[i].Name || !EdgesEqual(s1[i].Edges, s2[i].Edges) {
			return false
		}
	}
	return true
}

type NodesByID []dijkstrav2.Node

func (a NodesByID) Len() int           { return len(a) }
func (a NodesByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a NodesByID) Less(i, j int) bool { return a[i].ID < a[j].ID }

// 比较Edges
func EdgesEqual(s1, s2 []dijkstrav2.Edge) bool {
	sort.Sort(EdgesByID(s1))
	sort.Sort(EdgesByID(s2))
	if len(s1) != len(s2) {
		return false
	}
	for i := range s1 {
		if s1[i] != s2[i] {
			return false
		}
	}
	return true
}

type EdgesByID []dijkstrav2.Edge

func (a EdgesByID) Len() int           { return len(a) }
func (a EdgesByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a EdgesByID) Less(i, j int) bool { return a[i].To < a[j].To }

// 比较TargetNodes
func TargetNodesEqual(s1, s2 []dijkstrav2.TargetNode) bool {
	sort.Sort(TargetNodesByID(s1))
	sort.Sort(TargetNodesByID(s2))
	if len(s1) != len(s2) {
		return false
	}
	for i := range s1 {
		if s1[i] != s2[i] {
			return false
		}
	}
	return true
}

type TargetNodesByID []dijkstrav2.TargetNode

func (a TargetNodesByID) Len() int           { return len(a) }
func (a TargetNodesByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a TargetNodesByID) Less(i, j int) bool { return a[i].ID < a[j].ID }
