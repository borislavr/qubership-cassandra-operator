package robotTests

import (
	"github.com/Netcracker/qubership-cassandra-supplementary/pkg/utils"
	"github.com/Netcracker/qubership-nosqldb-operator-core/pkg/core"
)

type RobotCompound struct {
	core.MicroServiceCompound
}

type RobotBuilder struct {
	core.ExecutableBuilder
}

func (r *RobotBuilder) Build(ctx core.ExecutionContext) core.Executable {

	robot := RobotCompound{}
	robot.ServiceName = utils.Robot
	robot.CalcDeployType = func(ctx core.ExecutionContext) (deployType core.MicroServiceDeployType, err error) {
		return core.CleanDeploy, nil
	}
	robot.AddStep(&RobotDeployment{})

	return &robot
}
