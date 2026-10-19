package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/1Panel-dev/1Panel/backend/app/dto"
	"github.com/1Panel-dev/1Panel/backend/app/model"
	"github.com/1Panel-dev/1Panel/backend/global"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTestDBForService(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}
	if err := db.AutoMigrate(&model.HostNode{}); err != nil {
		t.Fatalf("failed to auto migrate: %v", err)
	}
	global.DB = db
}

func TestNodeService_CRUD(t *testing.T) {
	setupTestDBForService(t)

	// Mock sub-node health endpoint
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer mockServer.Close()

	svc := NewINodeService()

	// 1. Initial list should always include Master (local)
	list, err := svc.List()
	if err != nil {
		t.Fatalf("failed to list nodes: %v", err)
	}
	if len(list) != 1 || list[0].ID != "local" {
		t.Fatalf("expected 1 local master node, got %v", list)
	}

	// 2. Create a new worker node
	err = svc.Create(dto.NodeCreate{
		Name:        "Edge-Node-01",
		Address:     mockServer.URL,
		SecretToken: "test-secret-token",
		Description: "Primary edge worker",
	})
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}

	// 3. List again, should have 2 nodes
	list2, err := svc.List()
	if err != nil || len(list2) != 2 {
		t.Fatalf("expected 2 nodes, got %v (err: %v)", len(list2), err)
	}
	createdID := list2[1].ID

	// 4. Update node
	err = svc.Update(dto.NodeUpdate{
		ID:          createdID,
		Name:        "Edge-Node-01-Renamed",
		Address:     mockServer.URL,
		Description: "Updated description",
	})
	if err != nil {
		t.Fatalf("failed to update node: %v", err)
	}

	// 5. Query and verify update
	node, err := svc.GetNodeByID(createdID)
	if err != nil {
		t.Fatalf("failed to get node: %v", err)
	}
	if node.Name != "Edge-Node-01-Renamed" {
		t.Fatalf("expected Edge-Node-01-Renamed, got %s", node.Name)
	}

	// 6. Delete node
	err = svc.Delete(createdID)
	if err != nil {
		t.Fatalf("failed to delete node: %v", err)
	}

	list3, _ := svc.List()
	if len(list3) != 1 {
		t.Fatalf("expected 1 node after deletion, got %d", len(list3))
	}

	// 7. Prevent deleting local master
	errDelLocal := svc.Delete("local")
	if errDelLocal == nil {
		t.Fatalf("expected error when deleting local master node")
	}
}
