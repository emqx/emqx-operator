package api

import (
	"encoding/json"
	"fmt"
	"strconv"

	emperror "emperror.dev/errors"
	crd "github.com/emqx/emqx-operator/api/v2beta1"
	req "github.com/emqx/emqx-operator/internal/requester"
)

const ApiRebalanceV5 = "api/v5/load_rebalance"

type rebalanceStatus struct {
	Rebalances []crd.RebalanceState `json:"rebalances"`
}

func StartRebalance(req req.RequesterInterface, strategy crd.RebalanceStrategy, nodes []string) error {
	path := fmt.Sprintf("api/v5/load_rebalance/%s/start", nodes[0])
	request := map[string]interface{}{
		"conn_evict_rate":    strategy.ConnEvictRate,
		"sess_evict_rate":    strategy.SessEvictRate,
		"wait_takeover":      strategy.WaitTakeover,
		"wait_health_check":  strategy.WaitHealthCheck,
		"abs_conn_threshold": strategy.AbsConnThreshold,
		"abs_sess_threshold": strategy.AbsSessThreshold,
		"nodes":              nodes,
	}
	if len(strategy.RelConnThreshold) > 0 {
		relConnThreshold, _ := strconv.ParseFloat(strategy.RelConnThreshold, 64)
		request["rel_conn_threshold"] = relConnThreshold
	}
	if len(strategy.RelSessThreshold) > 0 {
		relSessThreshold, _ := strconv.ParseFloat(strategy.RelSessThreshold, 64)
		request["rel_sess_threshold"] = relSessThreshold
	}
	body, _ := json.Marshal(request)
	_, err := post(req, path, body)
	return err
}

func GetRebalanceStatus(req req.RequesterInterface) ([]crd.RebalanceState, error) {
	body, err := get(req, "api/v5/load_rebalance/global_status")
	if err != nil {
		return nil, err
	}
	rebalanceStatus := rebalanceStatus{}
	if err := json.Unmarshal(body, &rebalanceStatus); err != nil {
		return nil, emperror.Wrap(err, "unexpected rebalance status format")
	}
	return rebalanceStatus.Rebalances, nil
}

func StopRebalance(req req.RequesterInterface, coordinatorNode string) error {
	// stop rebalance should use coordinatorNode as path parameter
	path := fmt.Sprintf("api/v5/load_rebalance/%s/stop", coordinatorNode)
	_, err := post(req, path, nil)
	if err != nil {
		return err
	}
	return nil
}
