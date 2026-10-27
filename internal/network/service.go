package network

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type Flow struct {
	SourceIP      string `json:"source_ip"`
	DestinationIP string `json:"destination_ip"`
	Protocol      string `json:"protocol"`
	Bytes         int64  `json:"bytes"`
	Packets       int64  `json:"packets"`
}

type Summary struct {
	FlowCount int   `json:"flow_count"`
	Bytes     int64 `json:"bytes"`
	Packets   int64 `json:"packets"`
}

type Talker struct {
	IP      string `json:"ip"`
	Bytes   int64  `json:"bytes"`
	Packets int64  `json:"packets"`
	Flows   int    `json:"flows"`
}

type ProtocolStat struct {
	Protocol string `json:"protocol"`
	Flows    int    `json:"flows"`
	Bytes    int64  `json:"bytes"`
	Packets  int64  `json:"packets"`
}

type Service struct {
	path         string
	maxFileBytes int64
}

func NewService(path string, maxFileBytes int64) *Service {
	return &Service{path: path, maxFileBytes: maxFileBytes}
}

func (s *Service) Summary() (Summary, error) {
	flows, err := s.load()
	if err != nil {
		return Summary{}, err
	}
	var summary Summary
	for _, flow := range flows {
		summary.FlowCount++
		summary.Bytes += flow.Bytes
		summary.Packets += flow.Packets
	}
	return summary, nil
}

func (s *Service) TopTalkers(limit int) ([]Talker, error) {
	if limit < 1 || limit > 50 {
		return nil, fmt.Errorf("limit must be between 1 and 50")
	}
	flows, err := s.load()
	if err != nil {
		return nil, err
	}
	byIP := make(map[string]*Talker)
	for _, flow := range flows {
		for _, ip := range []string{flow.SourceIP, flow.DestinationIP} {
			talker := byIP[ip]
			if talker == nil {
				talker = &Talker{IP: ip}
				byIP[ip] = talker
			}
			talker.Bytes += flow.Bytes
			talker.Packets += flow.Packets
			talker.Flows++
		}
	}
	result := make([]Talker, 0, len(byIP))
	for _, talker := range byIP {
		result = append(result, *talker)
	}
	sortTalkers(result)
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (s *Service) ProtocolStatistics() ([]ProtocolStat, error) {
	flows, err := s.load()
	if err != nil {
		return nil, err
	}
	stats := make(map[string]*ProtocolStat)
	for _, flow := range flows {
		protocol := strings.ToUpper(flow.Protocol)
		stat := stats[protocol]
		if stat == nil {
			stat = &ProtocolStat{Protocol: protocol}
			stats[protocol] = stat
		}
		stat.Flows++
		stat.Bytes += flow.Bytes
		stat.Packets += flow.Packets
	}
	result := make([]ProtocolStat, 0, len(stats))
	for _, stat := range stats {
		result = append(result, *stat)
	}
	sortProtocols(result)
	return result, nil
}

func (s *Service) load() ([]Flow, error) {
	if strings.TrimSpace(s.path) == "" {
		return nil, fmt.Errorf("FLOW_CSV_PATH is not configured")
	}
	info, err := os.Lstat(s.path)
	if err != nil {
		return nil, fmt.Errorf("inspect flow CSV: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > s.maxFileBytes {
		return nil, fmt.Errorf("flow CSV must be a regular file no larger than %d bytes", s.maxFileBytes)
	}
	file, err := os.Open(s.path)
	if err != nil {
		return nil, fmt.Errorf("open flow CSV: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(io.LimitReader(file, s.maxFileBytes+1))
	reader.TrimLeadingSpace = true
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read flow CSV header: %w", err)
	}
	indices := make(map[string]int, len(header))
	for i, name := range header {
		indices[strings.ToLower(strings.TrimSpace(name))] = i
	}
	for _, required := range []string{"source_ip", "destination_ip", "protocol", "bytes", "packets"} {
		if _, ok := indices[required]; !ok {
			return nil, fmt.Errorf("flow CSV is missing required column %q", required)
		}
	}
	var flows []Flow
	for line := 2; ; line++ {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read flow CSV row %d: %w", line, err)
		}
		flow := Flow{
			SourceIP:      strings.TrimSpace(record[indices["source_ip"]]),
			DestinationIP: strings.TrimSpace(record[indices["destination_ip"]]),
			Protocol:      strings.TrimSpace(record[indices["protocol"]]),
		}
		if flow.SourceIP == "" || flow.DestinationIP == "" || flow.Protocol == "" {
			return nil, fmt.Errorf("flow CSV row %d contains an empty required value", line)
		}
		if flow.Bytes, err = strconv.ParseInt(strings.TrimSpace(record[indices["bytes"]]), 10, 64); err != nil || flow.Bytes < 0 {
			return nil, fmt.Errorf("flow CSV row %d has an invalid byte count", line)
		}
		if flow.Packets, err = strconv.ParseInt(strings.TrimSpace(record[indices["packets"]]), 10, 64); err != nil || flow.Packets < 0 {
			return nil, fmt.Errorf("flow CSV row %d has an invalid packet count", line)
		}
		flows = append(flows, flow)
	}
	return flows, nil
}
