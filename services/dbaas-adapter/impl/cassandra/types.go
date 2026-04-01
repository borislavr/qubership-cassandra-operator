package cassandra

type CassandraUpdateSettingsResult struct {
	SettingUpdateResult map[string]SettingUpdateResult `json:"settingUpdateResult"`
}

type CassandraUpdateSettingsRequest struct {
	CurrentSettings map[string]interface{} `json:"currentSettings"`
	NewSettings     map[string]interface{} `json:"newSettings"`
}

type SettingUpdateResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
