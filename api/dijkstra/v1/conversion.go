package v1

import (
	dijkstrav2 "jinli.io/crdshortestpath/api/dijkstra/v2"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

var _ conversion.Convertible = &KnownNodes{}

func (kn *KnownNodes) ConvertTo(dst conversion.Hub) error {
	v2kn := dst.(*dijkstrav2.KnownNodes)

	v2kn.ObjectMeta = kn.ObjectMeta
	v2kn.Spec.NodeIdentity = kn.Spec.NodeIdentity
	v2kn.Spec.CostUnit = kn.Spec.CostUnit
	for i := range kn.Spec.Nodes {
		node := dijkstrav2.Node{}
		node.ID = kn.Spec.Nodes[i].ID
		node.Name = kn.Spec.Nodes[i].Name
		for j := range kn.Spec.Nodes[i].Edges {
			edge := dijkstrav2.Edge(kn.Spec.Nodes[i].Edges[j])
			node.Edges = append(node.Edges, edge)
		}
		v2kn.Spec.Nodes = append(v2kn.Spec.Nodes, node)
	}
	v2kn.Status.LastUpdate = kn.Status.LastUpdate
	return nil
}

func (kn *KnownNodes) ConvertFrom(src conversion.Hub) error {
	v2kn := src.(*dijkstrav2.KnownNodes)

	kn.ObjectMeta = v2kn.ObjectMeta
	kn.Spec.NodeIdentity = v2kn.Spec.NodeIdentity
	kn.Spec.CostUnit = v2kn.Spec.CostUnit
	for i := range v2kn.Spec.Nodes {
		node := Node{}
		node.ID = v2kn.Spec.Nodes[i].ID
		node.Name = v2kn.Spec.Nodes[i].Name
		for j := range v2kn.Spec.Nodes[i].Edges {
			edge := Edge(v2kn.Spec.Nodes[i].Edges[j])
			node.Edges = append(node.Edges, edge)
		}
		kn.Spec.Nodes = append(kn.Spec.Nodes, node)
	}
	kn.Status.LastUpdate = v2kn.Status.LastUpdate
	return nil
}

var _ conversion.Convertible = &Display{}

func (dp *Display) ConvertTo(dst conversion.Hub) error {
	v2dp := dst.(*dijkstrav2.Display)

	v2dp.ObjectMeta = dp.ObjectMeta
	v2dp.Spec.NodeIdentity = dp.Spec.NodeIdentity
	v2dp.Spec.StartNode = dijkstrav2.StartNode(dp.Spec.StartNode)

	for i := range dp.Spec.TargetNodes {
		targetNode := dijkstrav2.TargetNode(dp.Spec.TargetNodes[i])
		v2dp.Spec.TargetNodes = append(v2dp.Spec.TargetNodes, targetNode)
	}

	if algorithm, ok := dp.Status.Record["algorithm"]; ok {
		v2dp.Spec.Algorithm = algorithm
	} else {
		v2dp.Spec.Algorithm = "dijkstra"
	}

	if cmStatus, ok := dp.Status.Record["computeStatus"]; ok {
		v2dp.Status.ComputeStatus = cmStatus
	} else {
		if dp.Status.LastUpdate.IsZero() {
			v2dp.Status.ComputeStatus = "Wait"
		} else {
			v2dp.Status.ComputeStatus = "Succeed"
		}
	}

	v2dp.Status.LastUpdate = dp.Status.LastUpdate

	return nil
}

func (dp *Display) ConvertFrom(src conversion.Hub) error {
	v2dp := src.(*dijkstrav2.Display)

	dp.ObjectMeta = v2dp.ObjectMeta
	dp.Spec.NodeIdentity = v2dp.Spec.NodeIdentity
	dp.Spec.StartNode = StartNode(v2dp.Spec.StartNode)

	for i := range dp.Spec.TargetNodes {
		targetNode := TargetNode(v2dp.Spec.TargetNodes[i])
		dp.Spec.TargetNodes = append(dp.Spec.TargetNodes, targetNode)
	}

	dp.Status.LastUpdate = v2dp.Status.LastUpdate
	record := make(map[string]string)
	record["algorithm"] = v2dp.Spec.Algorithm
	record["computeStatus"] = v2dp.Status.ComputeStatus
	dp.Status.Record = record
	return nil
}
