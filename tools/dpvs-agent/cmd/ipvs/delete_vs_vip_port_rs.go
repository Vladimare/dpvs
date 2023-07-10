package ipvs

import (
	"github.com/dpvs-agent/pkg/ipc/pool"
	"github.com/dpvs-agent/pkg/ipc/types"

	apiVs "github.com/dpvs-agent/restapi/operations/virtualserver"

	"github.com/go-openapi/runtime/middleware"
	"github.com/hashicorp/go-hclog"
)

type delVsRs struct {
	connPool *pool.ConnPool
	logger   hclog.Logger
}

func NewDelVsRs(cp *pool.ConnPool, parentLogger hclog.Logger) *delVsRs {
	logger := hclog.Default()
	if parentLogger != nil {
		logger = parentLogger.Named("DelVsVipPortRs")
	}
	return &delVsRs{connPool: cp, logger: logger}
}

func (h *delVsRs) Handle(params apiVs.DeleteVsVipPortRsParams) middleware.Responder {
	rs := types.NewRealServerFront()
	if err := rs.ParseVipPortProto(params.VipPort); err != nil {
		h.logger.Error("Convert to virtual server failed.", "VipPort", params.VipPort, "Error", err.Error())
		return apiVs.NewDeleteVsVipPortRsInvalidFrontend()
	}

	rss := make([]*types.RealServerSpec, len(params.Rss.Items))
	for i, s := range params.Rss.Items {
		rss[i] = types.NewRealServerSpec()
		rss[i].SetAf(rs.GetAf())
		rss[i].SetProto(rs.GetProto())
		rss[i].SetPort(s.Port)
		rss[i].SetAddr(s.IP)
		rss[i].SetWeight(uint32(s.Weight))
		// rss[i].SetConnFlags(types.DPVS_FWD_MODE_FNAT)
	}
	result := rs.Del(rss, h.connPool, h.logger)
	switch result {
	case types.EDPVS_OK:
		h.logger.Info("Del rs from virtual server success.", "VipPort", params.VipPort, "rss", rss)
		return apiVs.NewDeleteVsVipPortRsOK()
	case types.EDPVS_NOTEXIST:
		h.logger.Warn("There is some not exist rs with virtual server delete done.", "VipPort", params.VipPort, "rss", rss)
		return apiVs.NewDeleteVsVipPortRsOK()
	default:
		h.logger.Error("Del rs from virtual server failed.", "VipPort", params.VipPort, "rss", rss, "result", result.String())
	}

	return apiVs.NewDeleteVsVipPortRsFailure()
}
