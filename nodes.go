package main

type NodeList map[string]int

func GetUniqueNodes(aggs []HopsAgg) NodeList {
	uniques := make(NodeList)
	for _, agg := range aggs {
		incrementNodeList(uniques, agg.Source_ip)
		incrementNodeList(uniques, agg.Dest_ip)
	}
	return uniques
}

func incrementNodeList(nodeList NodeList, ip string) {
	if count, ok := nodeList[ip]; ok {
		nodeList[ip] = count + 1
	} else {
		nodeList[ip] = 1
	}
}
