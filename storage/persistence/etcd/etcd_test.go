package etcd

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func TestStorage_Base(t *testing.T) {
	endpoints := []string{"0.0.0.0:2379"}
	cli, _ := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	etcdStorage, _ := New(endpoints)
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	cli.Delete(ctx, "/", clientv3.WithPrefix())
	err := etcdStorage.CreateNamespace(context.Background(), "testNs")
	assert.Equal(t, nil, err)
	err = etcdStorage.CreateCluster(context.Background(), "testNs", "testCluster", nil)
	assert.Equal(t, "nil cluster info", err.Error())
	err = etcdStorage.RemoveCluster(context.Background(), "testNs", "testCluster")
	assert.Equal(t, nil, err)
	err = etcdStorage.RemoveNamespace("testNs", context.Background())
	assert.Equal(t, nil, err)
}

func GetFailoverTasks() []*FailOverTask {
	task1 := &FailOverTask{
		Namespace:  "testNs",
		Cluster:    "testCluster",
		ShardIdx:   0,
		Type:       1,
		ProbeCount: 2,
	}
	task2 := &FailOverTask{
		Namespace:  "testNs",
		Cluster:    "testCluster",
		ShardIdx:   1,
		Type:       1,
		ProbeCount: 2,
	}
	task3 := &FailOverTask{
		Namespace:  "testNs",
		Cluster:    "testCluster",
		ShardIdx:   2,
		Type:       0,
		ProbeCount: 2,
	}
	return []*FailOverTask{task1, task2, task3}
}

func TestStorage_Failover(t *testing.T) {
	endpoints := []string{"0.0.0.0:2379"}
	cli, _ := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	etcdStorage, _ := New(endpoints)
	tasks := GetFailoverTasks()
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	cli.Delete(ctx, "/"+tasks[0].Namespace+"/"+tasks[0].Cluster, clientv3.WithPrefix())

	err := etcdStorage.UpdateFailOverTask(context.Background(), tasks[0])
	assert.Equal(t, nil, err)
	task, _ := etcdStorage.GetFailOverTask(context.Background(), tasks[0].Namespace, tasks[0].Cluster)
	assert.Equal(t, 0, task.ShardIdx)
	err = etcdStorage.AddFailOverHistory(context.Background(), tasks[1])
	assert.Equal(t, nil, err)
	tasks, _ = etcdStorage.GetFailOverHistory(context.Background(), tasks[0].Namespace, tasks[0].Cluster)
	assert.Equal(t, 1, len(tasks))
	assert.Equal(t, 1, tasks[0].ShardIdx)
}

func GetMigTasks() []*MigrateTask {
	task1 := &MigrateTask{
		TaskID:      uint64(1),
		SubID:       uint64(1),
		Namespace:   "testNs",
		Cluster:     "testCluster",
		Source:      0,
		Target:      1,
		ErrorDetail: errors.New("failed").Error(),
	}
	task2 := &MigrateTask{
		TaskID:    uint64(1),
		SubID:     uint64(2),
		Namespace: "testNs",
		Cluster:   "testCluster",
		Source:    1,
		Target:    2,
	}
	task3 := &MigrateTask{
		TaskID:    uint64(2),
		SubID:     uint64(1),
		Namespace: "testNs",
		Cluster:   "testCluster",
		Source:    0,
		Target:    1,
	}
	return []*MigrateTask{task1, task2, task3}
}

func TestStorage_Migrate(t *testing.T) {
	endpoints := []string{"0.0.0.0:2379"}
	cli, _ := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	etcdStorage, _ := New(endpoints)
	tasks := GetMigTasks()
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	cli.Delete(ctx, "/"+tasks[0].Namespace+"/"+tasks[0].Cluster, clientv3.WithPrefix())

	etcdStorage.AddPendingMigrateTask(context.Background(), tasks[0].Namespace, tasks[0].Cluster, tasks)
	has, _ := etcdStorage.IsMigrateTaskExists(context.Background(), tasks[0].Namespace, tasks[0].Cluster, tasks[0].TaskID)
	assert.Equal(t, true, has)
	etcdStorage.RemovePendingMigrateTask(context.Background(), tasks[0])
	has, _ = etcdStorage.IsMigrateTaskExists(context.Background(), tasks[0].Namespace, tasks[0].Cluster, tasks[0].TaskID)
	assert.Equal(t, true, has)
	etcdStorage.RemovePendingMigrateTask(context.Background(), tasks[2])
	has, _ = etcdStorage.IsMigrateTaskExists(context.Background(), tasks[2].Namespace, tasks[2].Cluster, tasks[2].TaskID)
	assert.Equal(t, false, has)
	etcdStorage.AddMigrateTask(context.Background(), tasks[2])
	has, _ = etcdStorage.IsMigrateTaskExists(context.Background(), tasks[2].Namespace, tasks[2].Cluster, tasks[2].TaskID)
	assert.Equal(t, true, has)

	etcdStorage.AddMigrateHistory(context.Background(), tasks[0])
	etcdStorage.AddMigrateHistory(context.Background(), tasks[2])
	has, _ = etcdStorage.IsMigrateTaskExists(context.Background(), tasks[2].Namespace, tasks[2].Cluster, tasks[2].TaskID)
	assert.Equal(t, true, has)
	ts, _ := etcdStorage.GetMigrateHistory(context.Background(), tasks[0].Namespace, tasks[0].Cluster)
	assert.Equal(t, 2, len(ts))
	assert.Equal(t, ts[0].TaskID, tasks[0].TaskID)
	assert.Equal(t, ts[0].SubID, tasks[0].SubID)
	assert.Equal(t, ts[1].TaskID, tasks[2].TaskID)
	assert.Equal(t, ts[1].SubID, tasks[2].SubID)
}
