package rundeck

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rundeck/go-rundeck/rundeck"
)

var (
	_ resource.Resource                = &projectResource{}
	_ resource.ResourceWithConfigure   = &projectResource{}
	_ resource.ResourceWithImportState = &projectResource{}
)

func NewProjectResource() resource.Resource {
	return &projectResource{}
}

type projectResource struct {
	clients *RundeckClients
}

type projectResourceModel struct {
	ID                         types.String `tfsdk:"id"`
	Name                       types.String `tfsdk:"name"`
	Description                types.String `tfsdk:"description"`
	UIURL                      types.String `tfsdk:"ui_url"`
	ResourceModelSource        types.List   `tfsdk:"resource_model_source"`
	DefaultNodeFileCopierPlugin types.String `tfsdk:"default_node_file_copier_plugin"`
	DefaultNodeExecutorPlugin  types.String `tfsdk:"default_node_executor_plugin"`
	SSHAuthenticationType      types.String `tfsdk:"ssh_authentication_type"`
	SSHKeyStoragePath          types.String `tfsdk:"ssh_key_storage_path"`
	SSHKeyFilePath             types.String `tfsdk:"ssh_key_file_path"`
	ExtraConfig                types.Map    `tfsdk:"extra_config"`
}

type resourceModelSourceModel struct {
	Type   types.String `tfsdk:"type"`
	Config types.Map    `tfsdk:"config"`
}

func (r *projectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *projectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Rundeck project.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the project (same as name).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Unique name for the project.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "Description of the project to be shown in the Rundeck UI.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("Managed by Terraform"),
			},
			"ui_url": schema.StringAttribute{
				Description: "URL of the project in the Rundeck UI.",
				Computed:    true,
			},
			"resource_model_source": schema.ListAttribute{
				Description: "Resource model sources configuration.",
				Required:    true,
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"type":   types.StringType,
						"config": types.MapType{ElemType: types.StringType},
					},
				},
			},
			"default_node_file_copier_plugin": schema.StringAttribute{
				Description: "Default node file copier plugin.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("jsch-scp"),
			},
			"default_node_executor_plugin": schema.StringAttribute{
				Description: "Default node executor plugin.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("jsch-ssh"),
			},
			"ssh_authentication_type": schema.StringAttribute{
				Description: "SSH authentication type.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("privateKey"),
			},
			"ssh_key_storage_path": schema.StringAttribute{
				Description: "Path to SSH key in Rundeck key storage.",
				Optional:    true,
			},
			"ssh_key_file_path": schema.StringAttribute{
				Description: "Path to SSH key file on filesystem.",
				Optional:    true,
			},
			"extra_config": schema.MapAttribute{
				Description: "Additional raw configuration parameters to include in the project configuration, with dots replaced with slashes in the key names due to limitations in Terraform's config language.",
				ElementType: types.StringType,
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func (r *projectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	clients, ok := req.ProviderData.(*RundeckClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *RundeckClients, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.clients = clients
}

func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan projectResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.clients.V1
	apiCtx := context.Background()
	name := plan.Name.ValueString()

	// Check if project already exists
	project, _ := client.ProjectGet(apiCtx, name)
	if project.StatusCode != 404 {
		resp.Diagnostics.AddError(
			"Project already exists",
			fmt.Sprintf("Project with unique name (%s) already exists", name),
		)
		return
	}

	// Create bare minimum project
	_, err := client.ProjectCreate(apiCtx, rundeck.ProjectCreateRequest{
		Name: &name,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project",
			fmt.Sprintf("Could not create project: %s", err.Error()),
		)
		return
	}

	plan.ID = types.StringValue(name)

	// Now update with full configuration
	r.updateProjectConfig(ctx, apiCtx, client, name, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read back to get computed values
	r.readProject(ctx, apiCtx, client, name, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.clients.V1
	apiCtx := context.Background()
	name := state.ID.ValueString()

	r.readProject(ctx, apiCtx, client, name, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan projectResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.clients.V1
	apiCtx := context.Background()
	name := plan.ID.ValueString()

	// Update project configuration
	r.updateProjectConfig(ctx, apiCtx, client, name, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read back to ensure state is correct
	r.readProject(ctx, apiCtx, client, name, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state projectResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client := r.clients.V1
	apiCtx := context.Background()
	name := state.ID.ValueString()

	_, err := client.ProjectDelete(apiCtx, name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project",
			fmt.Sprintf("Could not delete project %s: %s", name, err.Error()),
		)
		return
	}
}

func (r *projectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}

// Helper function to update project configuration
func (r *projectResource) updateProjectConfig(ctx context.Context, apiCtx context.Context, client *rundeck.BaseClient, projectName string, plan *projectResourceModel, diags *diag.Diagnostics) {
	updateMap := map[string]string{}

	// Handle extra_config
	slashReplacer := strings.NewReplacer("/", ".")
	if !plan.ExtraConfig.IsNull() && !plan.ExtraConfig.IsUnknown() {
		extraConfig := make(map[string]types.String)
		diags.Append(plan.ExtraConfig.ElementsAs(ctx, &extraConfig, false)...)
		if diags.HasError() {
			return
		}
		for k, v := range extraConfig {
			updateMap[slashReplacer.Replace(k)] = v.ValueString()
		}
	}

	// Handle standard project attributes
	if !plan.Description.IsNull() {
		updateMap["project.description"] = plan.Description.ValueString()
	}
	if !plan.DefaultNodeFileCopierPlugin.IsNull() {
		updateMap["service.FileCopier.default.provider"] = plan.DefaultNodeFileCopierPlugin.ValueString()
	}
	if !plan.DefaultNodeExecutorPlugin.IsNull() {
		updateMap["service.NodeExecutor.default.provider"] = plan.DefaultNodeExecutorPlugin.ValueString()
	}
	if !plan.SSHAuthenticationType.IsNull() {
		updateMap["project.ssh-authentication"] = plan.SSHAuthenticationType.ValueString()
	}
	if !plan.SSHKeyStoragePath.IsNull() && !plan.SSHKeyStoragePath.IsUnknown() {
		updateMap["project.ssh-key-storage-path"] = plan.SSHKeyStoragePath.ValueString()
	}
	if !plan.SSHKeyFilePath.IsNull() && !plan.SSHKeyFilePath.IsUnknown() {
		updateMap["project.ssh-keypath"] = plan.SSHKeyFilePath.ValueString()
	}

	// Handle resource model sources
	var resourceModelSources []resourceModelSourceModel
	diags.Append(plan.ResourceModelSource.ElementsAs(ctx, &resourceModelSources, false)...)
	if diags.HasError() {
		return
	}

	for i, rms := range resourceModelSources {
		pluginType := rms.Type.ValueString()
		attrKeyPrefix := fmt.Sprintf("resources.source.%v.", i+1)
		typeKey := attrKeyPrefix + "type"
		configKeyPrefix := fmt.Sprintf("%vconfig.", attrKeyPrefix)
		updateMap[typeKey] = pluginType

		config := make(map[string]types.String)
		diags.Append(rms.Config.ElementsAs(ctx, &config, false)...)
		if diags.HasError() {
			return
		}
		for k, v := range config {
			updateMap[configKeyPrefix+k] = v.ValueString()
		}
	}

	_, err := client.ProjectConfigUpdate(apiCtx, projectName, updateMap)
	if err != nil {
		diags.AddError(
			"Error updating project configuration",
			fmt.Sprintf("Could not update project configuration: %s", err.Error()),
		)
		return
	}
}

// Helper function to read project state
func (r *projectResource) readProject(ctx context.Context, apiCtx context.Context, client *rundeck.BaseClient, name string, state *projectResourceModel, diags *diag.Diagnostics) {
	project, err := client.ProjectGet(apiCtx, name)
	if err != nil {
		diags.AddError(
			"Error reading project",
			fmt.Sprintf("Could not read project: %s", err.Error()),
		)
		return
	}

	if project.StatusCode == 404 {
		diags.AddError(
			"Project not found",
			fmt.Sprintf("Project %s not found", name),
		)
		return
	}

	projectConfig := project.Config.(map[string]interface{})

	// Set standard attributes
	for configKey, attrKey := range projectConfigAttributes {
		if v, ok := projectConfig[configKey]; ok {
			switch attrKey {
			case "name":
				state.Name = types.StringValue(v.(string))
			case "description":
				state.Description = types.StringValue(v.(string))
			case "default_node_file_copier_plugin":
				state.DefaultNodeFileCopierPlugin = types.StringValue(v.(string))
			case "default_node_executor_plugin":
				state.DefaultNodeExecutorPlugin = types.StringValue(v.(string))
			case "ssh_authentication_type":
				state.SSHAuthenticationType = types.StringValue(v.(string))
			case "ssh_key_storage_path":
				state.SSHKeyStoragePath = types.StringValue(v.(string))
			case "ssh_key_file_path":
				state.SSHKeyFilePath = types.StringValue(v.(string))
			}
			delete(projectConfig, configKey)
		}
	}

	// Parse resource model sources
	resourceSourceMap := map[int]interface{}{}
	configMaps := map[int]interface{}{}
	for configKey, v := range projectConfig {
		if strings.HasPrefix(configKey, "resources.source.") {
			nameParts := strings.Split(configKey, ".")
			if len(nameParts) < 4 {
				continue
			}

			index, err := strconv.Atoi(nameParts[2])
			if err != nil {
				continue
			}

			if _, ok := resourceSourceMap[index]; !ok {
				configMap := map[string]interface{}{}
				configMaps[index] = configMap
				resourceSourceMap[index] = map[string]interface{}{
					"config": configMap,
				}
			}

			switch nameParts[3] {
			case "type":
				if len(nameParts) != 4 {
					continue
				}
				m := resourceSourceMap[index].(map[string]interface{})
				m["type"] = v
			case "config":
				if len(nameParts) != 5 {
					continue
				}
				m := configMaps[index].(map[string]interface{})
				m[nameParts[4]] = v
			}

			delete(projectConfig, configKey)
		}
	}

	// Convert resource model sources to list
	resourceSourceIndices := []int{}
	for k := range resourceSourceMap {
		resourceSourceIndices = append(resourceSourceIndices, k)
	}
	sort.Ints(resourceSourceIndices)

	resourceModelSourceElements := []attr.Value{}
	for _, index := range resourceSourceIndices {
		source := resourceSourceMap[index].(map[string]interface{})
		
		configMap := make(map[string]attr.Value)
		if configInterface, ok := source["config"]; ok {
			for k, v := range configInterface.(map[string]interface{}) {
				configMap[k] = types.StringValue(v.(string))
			}
		}

		sourceModel := resourceModelSourceModel{
			Type:   types.StringValue(source["type"].(string)),
			Config: types.MapValueMust(types.StringType, configMap),
		}

		objType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"type":   types.StringType,
				"config": types.MapType{ElemType: types.StringType},
			},
		}

		objVal, diagsObj := types.ObjectValueFrom(ctx, objType.AttrTypes, sourceModel)
		diags.Append(diagsObj...)
		if diags.HasError() {
			return
		}

		resourceModelSourceElements = append(resourceModelSourceElements, objVal)
	}

	listType := types.ListType{
		ElemType: types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"type":   types.StringType,
				"config": types.MapType{ElemType: types.StringType},
			},
		},
	}

	listVal, diagsList := types.ListValue(listType.ElemType, resourceModelSourceElements)
	diags.Append(diagsList...)
	if diags.HasError() {
		return
	}
	state.ResourceModelSource = listVal

	// Handle extra_config
	extraConfig := map[string]attr.Value{}
	dotReplacer := strings.NewReplacer(".", "/")
	for k, v := range projectConfig {
		extraConfig[dotReplacer.Replace(k)] = types.StringValue(v.(string))
	}
	extraConfigMap, diagsMap := types.MapValue(types.StringType, extraConfig)
	diags.Append(diagsMap...)
	if diags.HasError() {
		return
	}
	state.ExtraConfig = extraConfigMap

	state.Name = types.StringValue(*project.Name)
	state.UIURL = types.StringValue(*project.URL)
}

