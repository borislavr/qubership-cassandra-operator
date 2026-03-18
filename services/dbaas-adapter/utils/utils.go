package utils

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Netcracker/qubership-dbaas-adapter-core/pkg/utils"

	"go.uber.org/zap"
)

var (
	log            *zap.Logger
	labelsFileName = flag.String(
		"labels_file_location_name",
		"dbaas.physical_databases.registration.labels.json",
		"File name where labels are located in json key-value format",
	)
	labelsLocationDir = flag.String(
		"labels_file_location_dir",
		"/app/config/",
		"Directory with file where labels are located in json key-value format",
	)
)

func GetCACert() string {
	cert, _ := os.LookupEnv("TLS_ROOTCERT")
	return cert
}

func GetEvAsStringSlice(name string, defaultValue []string) []string {
	value, exist := os.LookupEnv(name)
	if exist {
		value = value[1 : len(value)-1]
		return strings.Split(value, " ")
	}

	return defaultValue
}

func GetEnvBool(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		bvalue, err := strconv.ParseBool(value)
		if err != nil {
			log.Error(fmt.Sprintf("Can't parse %s boolean variable", key), zap.Error(err))
			panic(err)
		}
		return bvalue
	}
	return fallback
}

func ReadLabelsFile() map[string]string {
	logger := utils.GetLogger(GetEnvBool("LOG_DEBUG", false))
	file, err := os.ReadFile(*labelsLocationDir + *labelsFileName)
	if err != nil {
		logger.Warn(fmt.Sprintf("Skipping labels file, cannot read it: %s", *labelsLocationDir+*labelsFileName))
		return make(map[string]string)
	}
	var labels map[string]string
	err = json.Unmarshal(file, &labels)
	if err != nil {
		logger.Warn(fmt.Sprintf("Failed to parse labels file %s", *labelsLocationDir+*labelsFileName), zap.Error(err))
		labels = make(map[string]string)
	}
	logger.Info(fmt.Sprintf("Labels: %v", labels))
	return labels
}
