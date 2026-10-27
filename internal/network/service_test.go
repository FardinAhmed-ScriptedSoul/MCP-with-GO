package network

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyzeFlows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "flows.csv")
	csv := "source_ip,destination_ip,protocol,bytes,packets\n10.0.0.1,10.0.0.2,TCP,100,2\n10.0.0.2,10.0.0.3,UDP,50,1\n"
	if err := os.WriteFile(path, []byte(csv), 0600); err != nil {
		t.Fatal(err)
	}
	service := NewService(path, 1024)
	summary, err := service.Summary()
	if err != nil || summary != (Summary{FlowCount: 2, Bytes: 150, Packets: 3}) {
		t.Fatalf("Summary() = %+v, %v", summary, err)
	}
	talkers, err := service.TopTalkers(1)
	if err != nil || len(talkers) != 1 || talkers[0].IP != "10.0.0.2" || talkers[0].Bytes != 150 {
		t.Fatalf("TopTalkers() = %+v, %v", talkers, err)
	}
	protocols, err := service.ProtocolStatistics()
	if err != nil || len(protocols) != 2 || protocols[0].Protocol != "TCP" {
		t.Fatalf("ProtocolStatistics() = %+v, %v", protocols, err)
	}
}

func TestRejectInvalidFlowCSV(t *testing.T) {
	path := filepath.Join(t.TempDir(), "flows.csv")
	if err := os.WriteFile(path, []byte("src,dst\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewService(path, 1024).Summary(); err == nil {
		t.Fatal("expected missing column error")
	}
}
