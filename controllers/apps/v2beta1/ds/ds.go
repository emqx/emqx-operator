package v2beta1

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

func IsDSEnabled(r req.RequesterInterface) (bool, error) {
	_, err := apiGet(r, "api/v5/ds/sites")
	if err == nil {
		return true, nil
	}
	if emperror.Is(err, apiErrorNotFound) {
		return false, nil
	}
	return false, err
}

func GetReplicationStatus(r req.RequesterInterface) (DSReplicationStatus, error) {
	status := DSReplicationStatus{DBs: []DSDBReplicationStatus{}}

	body, err := apiGet(r, "api/v5/ds/storages")
	if err != nil {
		return status, err
	}

	var dsDatabases []string
	if err := json.Unmarshal(body, &dsDatabases); err != nil {
		return status, emperror.Wrap(err, "failed to retreive DS DBs")
	}

	dbStatus := DSDBReplicationStatus{}
	for _, db := range dsDatabases {
		body, err := apiGet(r, "api/v5/ds/storages/"+db)
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

func GetCluster(r req.RequesterInterface) (DSCluster, error) {
	cluster := DSCluster{Sites: []DSSite{}}

	body, err := apiGet(r, "api/v5/ds/sites")
	if err != nil {
		return cluster, err
	}

	var sites []string
	if err := json.Unmarshal(body, &sites); err != nil {
		return cluster, emperror.Wrap(err, "failed to retreive DS sites")
	}

	for _, s := range sites {
		site := DSSite{ID: s}
		body, err := apiGet(r, "api/v5/ds/sites/"+s)
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

func (c *DSCluster) FindSite(node string) *DSSite {
	for _, site := range c.Sites {
		if site.Node == node {
			return &site
		}
	}
	return nil
}

func UpdateReplicaSet(r req.RequesterInterface, db string, sites []string) error {
	body, _ := json.Marshal(sites)
	_, err := apiRequest(r, "PUT", "api/v5/ds/storages/"+db+"/replicas", body)
	return err
}

type apiError struct {
	StatusCode int
	Message    string
}

var (
	apiErrorNotFound = apiError{StatusCode: 404}
)

func (e apiError) Error() string {
	return fmt.Sprintf("HTTP %d, response: %s", e.StatusCode, e.Message)
}

func (e apiError) Is(target error) bool {
	if target, ok := target.(apiError); ok {
		return e.StatusCode == target.StatusCode
	}
	return false
}

func apiGet(r req.RequesterInterface, path string) ([]byte, error) {
	return apiRequest(r, "GET", path, nil)
}

func apiRequest(r req.RequesterInterface, method string, path string, body []byte) ([]byte, error) {
	url := r.GetURL(path)
	resp, body, err := r.Request(method, url, body, nil)
	if err != nil {
		return nil, emperror.Wrapf(err, "error accessing DS API %s", url.String())
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		err := apiError{StatusCode: resp.StatusCode, Message: string(body)}
		return nil, emperror.Wrapf(err, "error accessing DS API %s", url.String())
	}
	return body, nil
}
