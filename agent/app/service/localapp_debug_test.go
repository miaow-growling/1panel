package service

import (
	"testing"

	"github.com/1Panel-dev/1Panel/agent/app/model"
)

func TestDebugLocalAppDetail(t *testing.T) {
	d := model.AppDetail{}
	err := handleLocalAppDetail("/tmp/app-pkg/proxysql-admin/1.0.0", &d)
	if err != nil {
		t.Fatalf("handleLocalAppDetail err: %v", err)
	}
	t.Logf("compose len=%d, params=%s", len(d.DockerCompose), d.Params)
}

func TestDebugLocalApp(t *testing.T) {
	app, err := handleLocalApp("/tmp/app-pkg/proxysql-admin")
	if err != nil {
		t.Fatalf("handleLocalApp err: %v", err)
	}
	t.Logf("app key=%s name=%s type=%s", app.Key, app.Name, app.Type)
}
