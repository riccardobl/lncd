package main

import (
	"fmt"
	"time"
)

var (
	lastStats *Stats = nil
)

type ConnectionStats struct {
	NumPendingActions int
	Status            string
}

type Stats struct {
	NumConnections int
	Connections    []ConnectionStats
}

func refreshStats(pool *ConnectionPool, stats *Stats) *Stats {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()

	if stats == nil {
		stats = &Stats{
			NumConnections: 0,
			Connections:    nil,
		}
	}

	stats.NumConnections = len(pool.connections)
	if stats.Connections == nil || len(stats.Connections) != len(pool.connections) {
		stats.Connections = make([]ConnectionStats, len(pool.connections))
	}

	var i int
	for _, conn := range pool.connections {
		stats.Connections[i] = ConnectionStats{
			NumPendingActions: len(conn.actions),
			Status:            conn.connInfo.Status,
		}
		i++
	}

	return stats
}

func getStats() *Stats {
	return lastStats
}

func startStatsLoop(pool *ConnectionPool) {
	ticker := time.NewTicker(LNCD_STATS_INTERVAL)
	go func() {
		for range ticker.C {
			lastStats = refreshStats(pool, lastStats)

			if lastStats != nil {
				var statsString string = ""
				statsString += fmt.Sprintf("\nActive connections: %d", lastStats.NumConnections)
				for i, conn := range lastStats.Connections {
					statsString += fmt.Sprintf("\n    Connection id: %d", i)
					statsString += fmt.Sprintf("\n        Pending actions: %d", conn.NumPendingActions)
					statsString += fmt.Sprintf("\n        Status: %s", conn.Status)
				}
				log.Debugf("Stats: %s", statsString)
			}
		}
	}()
}
