package rundeck

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// buildScmSetupBody is a pure function; these are plain unit tests that
// don't need a live server to exercise it.

func TestBuildScmSetupBody_wrapsConfigKey(t *testing.T) {
	ctx := context.Background()

	config, diags := types.MapValue(types.StringType, map[string]attr.Value{
		"url": types.StringValue("git@example.com:org/repo.git"),
		"dir": types.StringValue(".rundeck-scm"),
	})
	if diags.HasError() {
		t.Fatalf("building test config: %v", diags)
	}

	body, diags := buildScmSetupBody(ctx, config)
	if diags.HasError() {
		t.Fatalf("buildScmSetupBody: %v", diags)
	}

	pluginConfig, ok := body["config"].(map[string]interface{})
	if !ok {
		t.Fatalf("body[\"config\"] = %T, want map[string]interface{}", body["config"])
	}
	if pluginConfig["url"] != "git@example.com:org/repo.git" {
		t.Errorf("url = %v, want the configured value", pluginConfig["url"])
	}
	if pluginConfig["dir"] != ".rundeck-scm" {
		t.Errorf("dir = %v, want the configured value", pluginConfig["dir"])
	}
}

// A null map value (e.g. `config = { pathTemplate = null }`, leaving an
// optional plugin property unset) must be omitted from the request body
// entirely, not sent as an empty string - the plugin may treat "" as an
// explicit, invalid value rather than "not provided".
func TestBuildScmSetupBody_omitsNullValues(t *testing.T) {
	ctx := context.Background()

	config, diags := types.MapValue(types.StringType, map[string]attr.Value{
		"url":          types.StringValue("git@example.com:org/repo.git"),
		"pathTemplate": types.StringNull(),
	})
	if diags.HasError() {
		t.Fatalf("building test config: %v", diags)
	}

	body, diags := buildScmSetupBody(ctx, config)
	if diags.HasError() {
		t.Fatalf("buildScmSetupBody: %v", diags)
	}

	pluginConfig := body["config"].(map[string]interface{})
	if _, exists := pluginConfig["pathTemplate"]; exists {
		t.Errorf("pathTemplate = %q, want it omitted when null", pluginConfig["pathTemplate"])
	}
	if _, exists := pluginConfig["url"]; !exists {
		t.Errorf("url missing, want it present")
	}
}

func TestBuildScmSetupBody_emptyConfig(t *testing.T) {
	ctx := context.Background()

	config, diags := types.MapValue(types.StringType, map[string]attr.Value{})
	if diags.HasError() {
		t.Fatalf("building test config: %v", diags)
	}

	body, diags := buildScmSetupBody(ctx, config)
	if diags.HasError() {
		t.Fatalf("buildScmSetupBody: %v", diags)
	}

	pluginConfig, ok := body["config"].(map[string]interface{})
	if !ok {
		t.Fatalf("body[\"config\"] = %T, want map[string]interface{}", body["config"])
	}
	if len(pluginConfig) != 0 {
		t.Errorf("pluginConfig = %v, want empty", pluginConfig)
	}
}
