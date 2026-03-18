package fiber

import (
	"context"
	"sync"
	"time"

	nosqlFiber "github.com/Netcracker/qubership-nosqldb-operator-core/pkg/fiber"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

var runFiberServerOnce sync.Once

func RunFiberServer(serverPort int, service CassandraFiberService, namespace string, log *zap.Logger) error {
	var err error
	runFiberServerOnce.Do(func() {
		err = (*nosqlFiber.GetFiberService()).Create(serverPort, func(app *fiber.App, serverContext context.Context) error {
			app.Server().ReadTimeout = time.Duration(60 * time.Second)
			app.Server().WriteTimeout = time.Duration(60 * time.Second)
			app.Server().IdleTimeout = time.Duration(60 * time.Second)
			SetCassandraOperatorHandlers(app, service, namespace, log)
			return nil
		}, true)
	})

	return err
}
