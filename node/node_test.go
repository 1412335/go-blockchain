package node

import (
	"context"
	"os"
	"path"
	"testing"
	"time"
)

func getTestDataDirPath() string {
	return path.Join(os.TempDir(), ".tbb")
}

func TestNode_Run(t *testing.T) {
	datadir := getTestDataDirPath()
	err := os.RemoveAll(datadir)
	if err != nil {
		t.Fatal(err)
	}

	n := New(datadir, "127.0.0.1", 8080, PeerNode{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := n.Run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer cancel()
}

func TestNode_Mining(t *testing.T) {
	
}
