package node

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/1412335/the-blockchain-bar/database"
)

func (n *Node) sync(ctx context.Context) error {
	ticker := time.NewTicker(45 * time.Second)

	for {
		select {
		case <-ticker.C:
			fmt.Println("Searching for new Peer and Block...")

			n.fetchNewBlocksAndPeers(ctx)

		case <-ctx.Done():
			ticker.Stop()
		}
	}
}

func (n *Node) fetchNewBlocksAndPeers(ctx context.Context) {
	for _, knownPeer := range n.knownPeers {
		if knownPeer.IP == n.ip && knownPeer.Port == n.port {
			continue
		}

		fmt.Printf("Searching for new Peers and their Blocks and Peers: '%s'\n", knownPeer.TCPAddress())

		status, err := queryPeerStatus(ctx, knownPeer)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			fmt.Printf("Peer '%s' was removed from KnownPeers\n", knownPeer.TCPAddress())

			n.RemovePeer(knownPeer)
			continue
		}

		if err := n.joinKnownPeers(ctx, knownPeer); err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		if err := n.syncBlocks(ctx, knownPeer, status); err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		n.syncKnownPeers(status.KnownPeers)

		if err := n.syncPendingTXs(knownPeer, status.PendingTxs); err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}
	}
}

// Add peer to node.knowPeers
func (n *Node) joinKnownPeers(ctx context.Context, peer PeerNode) error {
	if peer.connected {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://%s%s?ip=%s&port=%d", peer.TCPAddress(), endpointAddPeer, n.ip, n.port), nil)
	if err != nil {
		return err
	}

	r, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	rBodyJSON, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	addPeerRes := AddPeerRes{}
	if err := json.Unmarshal(rBodyJSON, &addPeerRes); err != nil {
		return err
	}
	if addPeerRes.Error != "" {
		return fmt.Errorf(addPeerRes.Error)
	}

	peer.connected = addPeerRes.Success
	n.AddPeer(peer)

	if !addPeerRes.Success {
		return fmt.Errorf("unable to join KnownPeers of '%s'", peer.TCPAddress())
	}
	return nil
}

// Fetch blocks from peer
func (n *Node) syncBlocks(ctx context.Context, peer PeerNode, status StatusRes) error {
	localBlockNumber := n.state.LatestBlock().Header.Number

	if status.Hash.IsEmpty() {
		return nil
	}

	if status.Number < localBlockNumber {
		return nil
	}

	if status.Number == 0 && !n.state.LatestBlockHash().IsEmpty() {
		return nil
	}

	newBlocksCount := status.Number - localBlockNumber
	if localBlockNumber == 0 && status.Number == 0 {
		newBlocksCount = 1
	}
	fmt.Printf("Found %d new blocks from Peer %s\n", newBlocksCount, peer.TCPAddress())

	if newBlocksCount == 0 {
		return nil
	}

	blocks, err := fetchBlocksFromPeer(ctx, peer, n.state.LatestBlockHash())
	if err != nil {
		return err
	}

	for _, block := range blocks {
		if _, err := n.state.AddBlock(block); err != nil {
			return err
		}
		// alert sync block & stop mining that block
		n.newSyncBlock <- block
	}
	return nil
}

// Sync node.knownPeers with peer.knownPeers
func (n *Node) syncKnownPeers(knownPeers map[string]PeerNode) {
	for _, peer := range knownPeers {
		if peer.IP == n.ip && peer.Port == n.port {
			continue
		}
		if _, isKnown := n.knownPeers[peer.TCPAddress()]; !isKnown {
			fmt.Printf("Found new Peer: %s\n", peer.TCPAddress())
			n.AddPeer(peer)
		}
	}
}

func (n *Node) syncPendingTXs(peer PeerNode, pendingTXs []database.TX) error {
	for _, tx := range pendingTXs {
		txHash, err := tx.Hash()
		if err != nil {
			return err
		}

		txJSON, err := json.Marshal(tx)
		if err != nil {
			return err
		}

		_, isPending := n.pendingTxs[txHash.Hex()]
		_, isArchived := n.archiveTxs[txHash.Hex()]

		if !isPending && !isArchived {
			fmt.Printf("Added Pending TX %s from Peer %s\n", txJSON, peer.TCPAddress())
			n.pendingTxs[txHash.Hex()] = tx
		}
	}
	return nil
}

func queryPeerStatus(ctx context.Context, peer PeerNode) (StatusRes, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://%s%s", peer.TCPAddress(), endpointStatus), nil)
	if err != nil {
		return StatusRes{}, err
	}

	r, err := http.DefaultClient.Do(req)
	if err != nil {
		return StatusRes{}, err
	}

	rBodyJSON, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return StatusRes{}, err
	}
	defer r.Body.Close()

	var statusRes StatusRes
	if err = json.Unmarshal(rBodyJSON, &statusRes); err != nil {
		return StatusRes{}, err
	}
	return statusRes, nil
}

func fetchBlocksFromPeer(ctx context.Context, peer PeerNode, hash database.Hash) ([]database.Block, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://%s%s?hash=%x", peer.TCPAddress(), endpointFetchBlocks, hash), nil)
	if err != nil {
		return nil, err
	}

	r, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	rBodyJSON, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	var statusRes FetchBlocksRes
	if err = json.Unmarshal(rBodyJSON, &statusRes); err != nil {
		return nil, err
	}
	return statusRes.Blocks, nil
}
