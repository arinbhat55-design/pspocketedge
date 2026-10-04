package docker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"

	agentv1 "github.com/ankitapaul1586-cmd/pspocketedge/internal/shared/pb/agentv1"
)

func fakeEngine(t *testing.T, version string, handler http.HandlerFunc) *client.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	cli, err := client.NewClientWithOpts(client.WithHost(server.URL), client.WithVersion(version))
	if err != nil {
		t.Fatal(err)
	}
	return cli
}

func TestPruneVolumesFilters(t *testing.T) {
	for _, tc := range []struct {
		name    string
		all     bool
		wantAll bool
	}{
		{"anonymous only", false, false},
		{"all unused", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got filters.Args
			cli := fakeEngine(t, "1.46", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/volumes/prune") {
					http.Error(w, r.URL.Path, http.StatusNotFound)
					return
				}
				var err error
				if got, err = filters.FromJSON(r.URL.Query().Get("filters")); err != nil {
					t.Error(err)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"VolumesDeleted": []string{"a", "b"}, "SpaceReclaimed": 2048})
			})

			report, err := PruneVolumes(context.Background(), cli, tc.all)
			if err != nil {
				t.Fatal(err)
			}
			if report.Removed != 2 || report.ReclaimedBytes != 2048 {
				t.Errorf("report = %+v, want 2 removed / 2048 bytes", report)
			}
			if !got.ExactMatch("label!", labelDeploymentID) {
				t.Errorf("filters %v must exclude deployment volumes", got)
			}
			if got.Contains("all") != tc.wantAll {
				t.Errorf("filters %v: all present = %v, want %v", got, got.Contains("all"), tc.wantAll)
			}
		})
	}
}

func TestPruneVolumesRefusesOldEngine(t *testing.T) {
	cli := fakeEngine(t, "1.40", func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected engine call %s %s", r.Method, r.URL.Path)
	})
	if _, err := PruneVolumes(context.Background(), cli, false); err == nil {
		t.Fatal("expected refusal on an engine that prunes named volumes by default")
	}
}

func TestPruneBuildCache(t *testing.T) {
	var gotAll string
	cli := fakeEngine(t, "1.46", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/build/prune") {
			http.Error(w, r.URL.Path, http.StatusNotFound)
			return
		}
		gotAll = r.URL.Query().Get("all")
		_ = json.NewEncoder(w).Encode(map[string]any{"SpaceReclaimed": 4096})
	})

	reclaimed, err := PruneBuildCache(context.Background(), cli, true)
	if err != nil {
		t.Fatal(err)
	}
	if reclaimed != 4096 || gotAll != "1" {
		t.Errorf("reclaimed = %d, all = %q; want 4096, \"1\"", reclaimed, gotAll)
	}
}

func TestContainerRemovePassesRemoveVolumes(t *testing.T) {
	for _, removeVolumes := range []bool{false, true} {
		var gotV string
		cli := fakeEngine(t, "1.46", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete {
				http.Error(w, r.URL.Path, http.StatusNotFound)
				return
			}
			gotV = r.URL.Query().Get("v")
			w.WriteHeader(http.StatusNoContent)
		})
		err := ContainerAction(context.Background(), cli, "abc", agentv1.ContainerAction_CONTAINER_ACTION_REMOVE, 0, true, removeVolumes)
		if err != nil {
			t.Fatal(err)
		}
		if want := map[bool]string{false: "", true: "1"}[removeVolumes]; gotV != want {
			t.Errorf("removeVolumes=%v: v = %q, want %q", removeVolumes, gotV, want)
		}
	}
}
