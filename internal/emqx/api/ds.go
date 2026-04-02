package api

import (
	"encoding/json"
	"fmt"

	emperror "emperror.dev/errors"
	req "github.com/emqx/emqx-operator/internal/requester"
)

// Describes how DS DBs are replicated across different sites.
type DSReplicationStatus struct {
	DBs []DSDBReplicationStatus
}

// Describes a single DS DB replication: which shards it has and how they are distributed across
// different sites.
type DSDBReplicationStatus struct {
	Name   string                     `json:"name"`
	Shards []DSShardReplicationStatus `json:"shards"`
}

type DSShardReplicationStatus struct {
	ID          string                    `json:"id"`
	Replicas    []DSShardReplicaStatus    `json:"replicas"`
	Transitions []DSShardTransitionStatus `json:"transitions,omitempty"`
}

type DSShardReplicaStatus struct {
	Site   string `json:"site"`
	Status string `json:"status"`
}

type DSShardTransitionStatus struct {
	Site       string `json:"site"`
	Transition string `json:"transition"`
}

// Describes a DS cluster.
type DSCluster struct {
	Sites []DSSite
}

// Describes a single site in the cluster.
type DSSite struct {
	ID     string
	Node   string              `json:"node"`
	Up     bool                `json:"up"`
	Shards []DSShardSiteStatus `json:"shards"`
}

type DSShardSiteStatus struct {
	DB         string `json:"storage"`
	ID         string `json:"id"`
	Status     string `json:"status,omitempty"`
	Transition string `json:"transition,omitempty"`
}

func (c *DSCluster) FindSite(node string) *DSSite {
	for _, site := range c.Sites {
		if site.Node == node {
			return &site
		}
	}
	return nil
}

func (s *DSReplicationStatus) TargetSites() (sites []string) {
	set := map[string]bool{}
	for _, db := range s.DBs {
		for _, shard := range db.Shards {
			for _, replica := range shard.Replicas {
				set[replica.Site] = true
			}
			for _, transition := range shard.Transitions {
				if transition.Transition == "joining" {
					set[transition.Site] = true
				}
				if transition.Transition == "leaving" {
					set[transition.Site] = false
				}
			}
		}
	}
	for site, included := range set {
		if included {
			sites = append(sites, site)
		}
	}
	return sites
}

func GetDSReplicationStatus(requester req.RequesterInterface) (DSReplicationStatus, error) {
	status := DSReplicationStatus{DBs: []DSDBReplicationStatus{}}

	body, err := get(requester, "api/v5/ds/storages")
	if err != nil {
		return status, err
	}

	var dsDatabases []string
	if err := json.Unmarshal(body, &dsDatabases); err != nil {
		return status, emperror.Wrap(err, "failed to retrieve DS DBs")
	}

	dbStatus := DSDBReplicationStatus{}
	for _, db := range dsDatabases {
		path := fmt.Sprintf("api/v5/ds/storages/%s", db)
		body, err := get(requester, path)
		if err != nil {
			return status, err
		}
		if err := json.Unmarshal(body, &dbStatus); err != nil {
			return status, emperror.Wrap(err, "failed to unmarshal DS DB replication status")
		}
		status.DBs = append(status.DBs, dbStatus)
	}
	return status, nil
}

func GetDSCluster(req req.RequesterInterface) (DSCluster, error) {
	cluster := DSCluster{Sites: []DSSite{}}

	body, err := get(req, "api/v5/ds/sites")
	if err != nil {
		return cluster, err
	}

	var sites []string
	if err := json.Unmarshal(body, &sites); err != nil {
		return cluster, emperror.Wrap(err, "failed to retrieve DS sites")
	}

	for _, s := range sites {
		site := DSSite{ID: s}
		path := fmt.Sprintf("api/v5/ds/sites/%s", s)
		body, err := get(req, path)
		if err != nil {
			return cluster, err
		}
		if err := json.Unmarshal(body, &site); err != nil {
			return cluster, emperror.Wrap(err, "failed to unmarshal DS site")
		}
		cluster.Sites = append(cluster.Sites, site)
	}

	return cluster, nil
}

func UpdateDSReplicaSet(requester req.RequesterInterface, db string, sites []string) error {
	path := fmt.Sprintf("api/v5/ds/storages/%s/replicas", db)
	body, _ := json.Marshal(sites)
	_, err := request(requester, "PUT", path, body, nil)
	return err
}

func ForgetDSSite(requester req.RequesterInterface, site string) error {
	path := fmt.Sprintf("api/v5/ds/sites/%s/forget", site)
	_, err := request(requester, "PUT", path, nil, nil)
	return err
}
