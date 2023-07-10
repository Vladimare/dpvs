package ipvs

import (
	"strings"

	// "github.com/dpvs-agent/models"
	"github.com/dpvs-agent/pkg/ipc/pool"
	"github.com/dpvs-agent/pkg/ipc/types"
	"golang.org/x/sys/unix"

	apiVs "github.com/dpvs-agent/restapi/operations/virtualserver"

	"github.com/go-openapi/runtime/middleware"
	"github.com/hashicorp/go-hclog"
)

type putVsItem struct {
	connPool *pool.ConnPool
	logger   hclog.Logger
}

func NewPutVsItem(cp *pool.ConnPool, parentLogger hclog.Logger) *putVsItem {
	logger := hclog.Default()
	if parentLogger != nil {
		logger = parentLogger.Named("PutVsVipPort")
	}
	return &putVsItem{connPool: cp, logger: logger}
}

// ipvsadm -A vip:port -s wrr
func (h *putVsItem) Handle(params apiVs.PutVsVipPortParams) middleware.Responder {
	vs := types.NewVirtualServerSpec()
	err := vs.ParseVipPortProto(params.VipPort)
	if err != nil {
		h.logger.Error("Convert to virtual server failed", "VipPort", params.VipPort, "Error", err.Error())
		return apiVs.NewPutVsVipPortInvalidFrontend()
	}

	schedName := ""

	if params.Spec != nil {
		schedName = params.Spec.SchedName

		vs.SetFwmark(params.Spec.Fwmark)
		vs.SetConnTimeout(params.Spec.ConnTimeout) // establish time out
		vs.SetBps(params.Spec.Bps)
		vs.SetLimitProportion(params.Spec.LimitProportion)

		if params.Spec.Timeout != 0 {
			vs.SetTimeout(params.Spec.Timeout) // persistence time out
			vs.SetFlagsPersistent()
		}

		if params.Spec.ExpireQuiescent != nil && *params.Spec.ExpireQuiescent {
			vs.SetFlagsExpireQuiescent()
		}

		if params.Spec.SyncProxy != nil && *params.Spec.SyncProxy {
			vs.SetFlagsSyncProxy()
		}
	}

	vs.SetSchedName(schedName)
	if strings.EqualFold(vs.GetSchedName(), "conhash") {
		vs.SetFlagsHashSrcIP()

		if vs.GetProto() == unix.IPPROTO_UDP {
			// if strings.EqualFold(strings.ToLower(params.Spec.HashTaget), "qid") {vs.SetFlagsHashQuicID()}
		}
	}

	result := vs.Add(h.connPool, h.logger)
	h.logger.Info("Add virtual server done.", "vs", vs, "result", result.String())
	switch result {
	case types.EDPVS_OK:
		// return 201
		h.logger.Info("Created new virtual server success.", "VipPort", params.VipPort)
		return apiVs.NewPutVsVipPortCreated()
	case types.EDPVS_EXIST:
		h.logger.Info("The virtual server already exist! Try to update.", "VipPort", params.VipPort)
		reason := vs.Update(h.connPool, h.logger)
		if reason != types.EDPVS_OK {
			// return 461
			h.logger.Error("Update virtual server failed.", "VipPort", params.VipPort, "reason", reason.String())
			return apiVs.NewPutVsVipPortInvalidBackend()
		}
		h.logger.Info("Update virtual server success.", "VipPort", params.VipPort)
		// return 200
		return apiVs.NewPutVsVipPortOK()
	default:
		h.logger.Error("Add virtual server failed.", "result", result.String())
		return apiVs.NewPutVsVipPortInvalidBackend()
	}

	return apiVs.NewPutVsVipPortOK()
}
