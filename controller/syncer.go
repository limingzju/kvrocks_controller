package controller

import (
	"context"
	"fmt"
	"sync"

	"github.com/KvrocksLabs/kvrocks_controller/logger"
	"go.uber.org/zap"

	"github.com/KvrocksLabs/kvrocks_controller/metadata"
	"github.com/KvrocksLabs/kvrocks_controller/storage"
	"github.com/KvrocksLabs/kvrocks_controller/util"
)

// Syncer would sync the cluster topology information
// to cluster nodes when it's changed.
type Syncer struct {
	storage  *storage.Storage
	wg       sync.WaitGroup
	shutdown chan struct{}
	notifyCh chan storage.Event
}

func NewSyncer(s *storage.Storage) *Syncer {
	syncer := &Syncer{
		storage:  s,
		shutdown: make(chan struct{}, 0),
		notifyCh: make(chan storage.Event, 8),
	}
	go syncer.loop()
	return syncer
}

func (syncer *Syncer) Notify(event *storage.Event) {
	syncer.notifyCh <- *event
}

func (syncer *Syncer) handleEvent(event *storage.Event) error {
	switch event.Type {
	case storage.EventCluster, storage.EventShard, storage.EventNode:
		return syncer.handleClusterEvent(event)
	default:
		return nil
	}
}

func (syncer *Syncer) handleClusterEvent(event *storage.Event) error {
	if event.Command != storage.CommandRemove {
		cluster, err := syncer.storage.GetClusterInfo(event.Namespace, event.Cluster)
		if err != nil {
			return fmt.Errorf("failed to get cluster: %w", err)
		}
		return syncClusterInfoToAllNodes(context.Background(), &cluster)
	}
	// TODO: Remove related cluster tasks
	return nil
}

func (syncer *Syncer) loop() {
	defer syncer.wg.Done()
	syncer.wg.Add(1)
	for {
		select {
		case event := <-syncer.notifyCh:
			if err := syncer.handleEvent(&event); err != nil {
				logger.Get().With(
					zap.Error(err),
					zap.Any("event", event),
				).Error("Failed to handle event")
			}
		case <-syncer.shutdown:
			return
		}
	}
}

func (syncer *Syncer) Close() {
	close(syncer.shutdown)
	close(syncer.notifyCh)
	syncer.wg.Wait()
}

func syncClusterInfoToNode(ctx context.Context, node *metadata.NodeInfo, clusterSlotsStr string, version int64) error {
	cli, err := util.NewRedisClient(node.Address)
	if err != nil {
		return fmt.Errorf("addr: %s, dail: %w", node.Address, err)
	}

	err = cli.Do(ctx, "CLUSTERX", "setnodeid", node.ID).Err()
	if err != nil {
		return fmt.Errorf("addr: %s, set node id: %w", node.Address, err)
	}
	err = cli.Do(ctx, "CLUSTERX", "setnodes", clusterSlotsStr, version).Err()
	if err != nil {
		return fmt.Errorf("addr: %s, set nodes: %w", node.Address, err)
	}
	return nil
}

func syncClusterInfoToAllNodes(ctx context.Context, cluster *metadata.Cluster) error {
	// FIXME: should keep retry in separate routine to prevent occurring error
	// and cause update failure.
	clusterSlotsStr, err := cluster.ToSlotString()
	if err != nil {
		return err
	}
	var errs []error
	for _, shard := range cluster.Shards {
		for _, node := range shard.Nodes {
			if err := syncClusterInfoToNode(ctx, &node, clusterSlotsStr, cluster.Version); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if errs != nil {
		return fmt.Errorf("%v", errs)
	}
	return nil
}
