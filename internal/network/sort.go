package network

import "sort"

func sortTalkers(talkers []Talker) {
	sort.Slice(talkers, func(i, j int) bool {
		if talkers[i].Bytes == talkers[j].Bytes {
			return talkers[i].IP < talkers[j].IP
		}
		return talkers[i].Bytes > talkers[j].Bytes
	})
}

func sortProtocols(protocols []ProtocolStat) {
	sort.Slice(protocols, func(i, j int) bool { return protocols[i].Protocol < protocols[j].Protocol })
}
