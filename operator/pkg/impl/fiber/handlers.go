package fiber

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	mUtils "github.com/Netcracker/qubership-cassandra-operator/pkg/impl/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type CassandraOperatorHandler struct {
	service CassandraFiberService
	logger  *zap.Logger
}

// FlushInMemoryData godoc
// @Tags Operator API
// @Summary Flus In-Memory Data
// @Description Flush In-Memory data into files
// @Produce  json
// @Failure 500 {string} Token "Internal error"
// @Param appName path string true "Application name"
// @Router /flush [post]
func (h *CassandraOperatorHandler) FlushInMemoryData(c *fiber.Ctx) error {
	err := c.JSON(h.service.FlushInMemoryData())
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	return c.SendStatus(http.StatusOK)
}

func calculateMetrics(operationResultMetric *prometheus.GaugeVec, service CassandraFiberService, namespace string) {
	for {
		operationResultMetric.WithLabelValues(namespace, "insert").Set(float64(service.PerformInsertTest()))
		operationResultMetric.WithLabelValues(namespace, "select").Set(float64(service.PerformSelectTest()))
		operationResultMetric.WithLabelValues(namespace, "delete").Set(float64(service.PerformDeleteTest()))

		time.Sleep(20 * time.Second)
	}
}

func SetCassandraOperatorHandlers(app *fiber.App, service CassandraFiberService, namespace string, logger *zap.Logger) {
	handler := &CassandraOperatorHandler{service: service, logger: logger}

	recoverConfig := recover.ConfigDefault
	recoverConfig.EnableStackTrace = true
	recoverConfig.StackTraceHandler = func(c *fiber.Ctx, e interface{}) {
		logger.Error(fmt.Sprintf("Panic: %+v\nStacktrace:\n%s", e, string(debug.Stack())))
	}
	app.Use(recover.New(recoverConfig))
	app.Use(func(c *fiber.Ctx) error {
		// Setting defaults for existed handlers
		c.Request().Header.SetContentType(utils.GetMIME("json"))
		logger.Debug(fmt.Sprintf("%s %s", c.Request().Header.Method(), c.Path()))
		return c.Next()
	})

	var operationResultMetric = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cassandra_smoke_test",
			Help: "metric exposed using the prometheus.",
		},
		[]string{"namespace", "operation"},
	)

	go calculateMetrics(operationResultMetric, service, namespace)

	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	app.Post(fmt.Sprintf("/%s", mUtils.FlushURI), handler.FlushInMemoryData)
}
