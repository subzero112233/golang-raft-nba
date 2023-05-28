package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"

	raftAPI "github.com/subzero112233/golang-raft-nba/api/raft"
	raftServer "github.com/subzero112233/golang-raft-nba/api/raft/server"
)

// GameFSM is an implementation of the Raft FSM interface
type GameFSM struct {
	mu     sync.Mutex
	Events []raftServer.Event
}

// Apply applies a Raft log entry to the FSM
func (f *GameFSM) Apply(logEntry *raft.Log) interface{} {
	var event raftServer.Event
	err := json.Unmarshal(logEntry.Data, &event)
	if err != nil {
		// Handle the error
		return nil
	}
	fmt.Printf("Received committed log entry: %v\n", event)
	// Process the committed log entry

	f.mu.Lock()
	defer f.mu.Unlock()

	f.Events = append(f.Events, event)
	fmt.Printf("Applied Raft log entry: %+v\n", event)

	return nil
}

// Snapshot returns a snapshot of the FSM state
func (f *GameFSM) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Create a snapshot of the game state
	snapshot := make([]raftServer.Event, len(f.Events))
	copy(snapshot, f.Events)

	return &GameSnapshot{Snapshot: snapshot}, nil
}

// Restore restores the FSM state from a snapshot
func (f *GameFSM) Restore(snapshot io.ReadCloser) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	b := new(bytes.Buffer)
	_, err := io.Copy(b, snapshot)
	if err != nil {
		return err
	}

	return json.Unmarshal(b.Bytes(), &f.Events)
}

// GameSnapshot is an implementation of the Raft FSMSnapshot interface
type GameSnapshot struct {
	Snapshot []raftServer.Event
}

// Persist saves the snapshot to the provided sink
func (s *GameSnapshot) Persist(sink raft.SnapshotSink) error {
	b, err := json.Marshal(s.Snapshot)
	if err != nil {
		return err
	}

	_, err = sink.Write(b)
	if err != nil {
		sink.Cancel()
		return fmt.Errorf("failed to sink write with error: %s", err.Error())
	}

	return sink.Close()
}

// Release releases any resources held by the snapshot
func (s *GameSnapshot) Release() {
	// Release any resources held by the snapshot
}

// main starts the application
func main() {
	port, ok := os.LookupEnv("PORT")
	if !ok {
		log.Fatal("PORT environment variable not set")
	}

	dataDir := filepath.Join("data", port)

	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		err := os.MkdirAll(dataDir, 0744)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Setup Raft configuration
	raftConfig := raft.DefaultConfig()
	raftConfig.LocalID = raft.ServerID(port) // since the port has to be unique, this is fine as an id.

	// Create a Raft store
	store, err := raftboltdb.NewBoltStore(filepath.Join(dataDir, "raft.db"))
	if err != nil {
		log.Fatal(err)
	}

	// Create transport for Raft communication
	addr, err := net.ResolveTCPAddr("tcp", "localhost:"+port)
	if err != nil {
		log.Fatal(err)
	}

	fss, err := raft.NewFileSnapshotStore(dataDir, 3, os.Stderr)
	if err != nil {
		log.Fatal(err)
	}

	transport, err := raft.NewTCPTransport(addr.String(), addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		log.Fatal(err)
	}

	// Create the Raft FSM
	fsm := &GameFSM{}

	// Create the Raft node
	raftNode, err := raft.NewRaft(raftConfig, fsm, store, store, fss, transport)
	if err != nil {
		log.Fatal(err)
	}

	if _, ok := os.LookupEnv("BOOTSTRAP_CLUSTER"); ok {
		cfg := raft.Configuration{
			Servers: []raft.Server{
				{
					Suffrage: raft.Voter,
					ID:       raft.ServerID(port),
					Address:  raft.ServerAddress("localhost:" + port),
				},
			},
		}
		f := raftNode.BootstrapCluster(cfg)
		if err := f.Error(); err != nil {
			log.Fatalf("WARNING, bootstrap failed with error: %s", err.Error())
		}
	}

	// use different ports for the API but keep the same convention
	portInteger, err := strconv.Atoi(port)
	if err != nil {
		log.Fatal(err)
	}

	raftAPIPort := portInteger + 100

	// Start an HTTP server for accepting client requests
	raftAPI.StartServer(raftNode, strconv.Itoa(raftAPIPort))
}
