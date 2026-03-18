package backup

import (
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/core"
)

type BackupConfigMaps struct {
	core.DefaultExecutable
}

func (r *BackupConfigMaps) Execute(ctx core.ExecutionContext) error {
	//request := ctx.Get(constants.ContextRequest).(reconcile.Request)
	//log := ctx.Get(constants.ContextLogger).(*zap.Logger)
	//
	//log.Info("Backup Config Maps creation step started")
	//
	//cfg := &v1.ConfigMap{
	//	ObjectMeta: v12.ObjectMeta{
	//		Namespace: request.Namespace,
	//		Name:      utils.BackupMonitoringConfig,
	//	},
	//	Data: map[string]string{
	//		"url.health": fmt.Sprintf("http://%s:8080/health", ctx.Get(utils.MonitoringIPTemplate).(string)),
	//	},
	//}
	//
	//err := utils.CreateRuntimeObjectContextWrapper(ctx, cfg, cfg.ObjectMeta)
	//
	//if err != nil {
	//	return err
	//}
	//
	//log.Info("Backup Monitoring Config has been created")

	return nil
}
