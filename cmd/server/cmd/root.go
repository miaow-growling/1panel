package cmd

import (
	"fmt"
	"os"
	"os/user"
	"strings"
	"time"

	"github.com/1Panel-dev/1Panel/backend/server"
	cmdUtils "github.com/1Panel-dev/1Panel/backend/utils/cmd"
	"github.com/glebarez/sqlite"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

var (
	language   string
	isAgent    bool
	agentToken string
)

func init() {
	RootCmd.PersistentFlags().StringVarP(&language, "language", "l", "en", "Set the language")
	RootCmd.PersistentFlags().BoolVar(&isAgent, "agent", false, "Run 1Panel in lightweight Worker Agent mode")
	RootCmd.PersistentFlags().StringVar(&agentToken, "token", "", "Secret token for Agent authentication")
}

var RootCmd = &cobra.Command{
	Use: "1panel",
	RunE: func(cmd *cobra.Command, args []string) error {
		if isAgent {
			_ = os.Setenv("PANEL_AGENT_MODE", "true")
		}
		if agentToken != "" {
			_ = os.Setenv("PANEL_AGENT_TOKEN", agentToken)
		}
		server.Start()
		return nil
	},
}

type setting struct {
	ID        uint      `gorm:"primarykey;AUTO_INCREMENT" json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Key       string    `json:"key" gorm:"type:varchar(256);not null;"`
	Value     string    `json:"value" gorm:"type:varchar(256)"`
	About     string    `json:"about" gorm:"type:longText"`
}

func loadDBConn() (*gorm.DB, error) {
	baseDir := "./data"
	if _, err := os.Stat("/usr/local/bin/1pctl"); err == nil {
		stdout, err := cmdUtils.Exec("grep '^BASE_DIR=' /usr/local/bin/1pctl | cut -d'=' -f2")
		if err == nil {
			trimmed := strings.ReplaceAll(stdout, "\n", "")
			if len(trimmed) > 0 {
				baseDir = trimmed
			}
		}
	}
	if strings.HasSuffix(baseDir, "/") {
		baseDir = baseDir[:strings.LastIndex(baseDir, "/")]
	}

	db, err := gorm.Open(sqlite.Open(baseDir+"/1panel/db/1Panel.db"), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("init my db conn failed, err: %v \n", err)
	}
	return db, nil
}

func getSettingByKey(db *gorm.DB, key string) string {
	var setting setting
	_ = db.Where("key = ?", key).First(&setting).Error
	return setting.Value
}

type LoginLog struct{}

func isDefault(db *gorm.DB) bool {
	logCount := int64(0)
	_ = db.Model(&LoginLog{}).Where("status = ?", "Success").Count(&logCount).Error
	return logCount == 0
}

func setSettingByKey(db *gorm.DB, key, value string) error {
	return db.Model(&setting{}).Where("key = ?", key).Updates(map[string]interface{}{"value": value}).Error
}

func isRoot() bool {
	currentUser, err := user.Current()
	if err != nil {
		return false
	}
	return currentUser.Uid == "0"
}
