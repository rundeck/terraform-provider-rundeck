package rundeck

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	openapi "github.com/rundeck/go-rundeck/rundeck-v2"
)

func resourceRundeckProjectRunner() *schema.Resource {
	return &schema.Resource{
		Create: CreateProjectRunner,
		Update: UpdateProjectRunner,
		Delete: DeleteProjectRunner,
		Read:   ReadProjectRunner,
		Importer: &schema.ResourceImporter{
			State: resourceProjectRunnerImport,
		},

		Schema: map[string]*schema.Schema{
			"project_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the project where the runner will be created",
			},

			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the runner",
			},

			"description": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Description of the runner",
			},

			"tag_names": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Comma separated tags for the runner",
			},

			"installation_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "linux",
				Description: "Installation type of the runner (linux, windows, kubernetes, docker)",
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					v := val.(string)
					validTypes := []string{"linux", "windows", "kubernetes", "docker"}
					for _, validType := range validTypes {
						if v == validType {
							return
						}
					}
					errs = append(errs, fmt.Errorf("%q must be one of %v, got: %s", key, validTypes, v))
					return
				},
			},

			"replica_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "manual",
				Description: "Replica type of the runner (manual or ephemeral)",
				ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
					v := val.(string)
					if v != "manual" && v != "ephemeral" {
						errs = append(errs, fmt.Errorf("%q must be either 'manual' or 'ephemeral', got: %s", key, v))
					}
					return
				},
			},

			// Node dispatch configuration
			"runner_as_node_enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Enable the runner to act as a node",
			},

			"remote_node_dispatch": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Enable remote node dispatch for the runner",
			},

			"runner_node_filter": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Node filter string for the runner",
			},

			// Read-only computed fields
			"runner_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ID of the created runner",
			},

			"token": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   false,
				Description: "Authentication token for the runner",
			},

			"download_token": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   false,
				Description: "Download token for the runner package",
			},
		},
	}
}

func CreateProjectRunner(d *schema.ResourceData, meta interface{}) error {
	clients := meta.(*RundeckClients)
	client := clients.V2
	ctx := clients.ctx

	projectName := d.Get("project_name").(string)
	name := d.Get("name").(string)
	description := d.Get("description").(string)

	// Create the runner request
	runnerRequest := openapi.NewCreateRunnerRequest(name, description)

	if tagNames, ok := d.GetOk("tag_names"); ok {
		tagNamesStr := tagNames.(string)
		runnerRequest.SetTagNames(tagNamesStr)
	}

	// Always set installation_type, using default "linux" if not specified
	installationType := d.Get("installation_type").(string)
	if installationType == "" {
		installationType = "linux"
	}
	runnerRequest.SetInstallationType(installationType)

	// Always set replica_type, using default "manual" if not specified
	replicaType := d.Get("replica_type").(string)
	if replicaType == "" {
		replicaType = "manual"
	}
	runnerRequest.SetReplicaType(replicaType)

	// Create the project runner request wrapper
	projectRunnerRequest := openapi.NewCreateProjectRunnerRequest(name, description)
	projectRunnerRequest.SetNewRunnerRequest(*runnerRequest)

	// Debug output
	log.Printf("[DEBUG] Creating project runner for project: %s", projectName)
	log.Printf("[DEBUG] Runner name: %s, description: %s", name, description)

	// Create the runner within the project context
	response, resp, err := client.RunnerAPI.CreateProjectRunner(ctx, projectName).CreateProjectRunnerRequest(*projectRunnerRequest).Execute()
	if err != nil {
		return fmt.Errorf("failed to create project runner: %v", err)
	}

	// Debug output
	log.Printf("[DEBUG] Create Project Runner API response status: %d", resp.StatusCode)
	if response != nil {
		log.Printf("[DEBUG] Create Project Runner response: %+v", response)
	}

	// Check if we have a runner ID to proceed with node dispatch configuration
	if response.RunnerId == nil {
		return fmt.Errorf("runner creation successful but no runner ID returned")
	}

	runnerId := *response.RunnerId

	// Configure node dispatch settings if any node dispatch options are specified
	if d.HasChange("runner_as_node_enabled") || d.HasChange("remote_node_dispatch") || d.HasChange("runner_node_filter") ||
		d.Get("runner_as_node_enabled").(bool) || d.Get("remote_node_dispatch").(bool) || d.Get("runner_node_filter").(string) != "" {

		// Create node dispatch configuration request
		nodeDispatchRequest := openapi.NewSaveProjectRunnerNodeDispatchSettingsRequest(runnerId)

		if runnerAsNodeEnabled, ok := d.GetOk("runner_as_node_enabled"); ok {
			nodeDispatchRequest.SetRunnerAsNodeEnabled(runnerAsNodeEnabled.(bool))
		}

		if remoteNodeDispatch, ok := d.GetOk("remote_node_dispatch"); ok {
			nodeDispatchRequest.SetRemoteNodeDispatch(remoteNodeDispatch.(bool))
		}

		if runnerNodeFilter, ok := d.GetOk("runner_node_filter"); ok {
			nodeDispatchRequest.SetRunnerNodeFilter(runnerNodeFilter.(string))
		}

		// Debug output
		log.Printf("[DEBUG] Configuring node dispatch for runner: %s in project: %s", runnerId, projectName)

		// Make the node dispatch configuration API call
		dispatchResp, dispatchHttpResp, err := client.RunnerAPI.SaveProjectRunnerNodeDispatchSettings(ctx, projectName).SaveProjectRunnerNodeDispatchSettingsRequest(*nodeDispatchRequest).Execute()
		if err != nil {
			// If node dispatch configuration fails, we should still proceed since the runner was created successfully
			log.Printf("[WARN] Failed to configure node dispatch for runner %s: %v", runnerId, err)
		} else {
			// Debug output
			log.Printf("[DEBUG] Node dispatch configuration API response status: %d", dispatchHttpResp.StatusCode)
			if dispatchResp != nil {
				log.Printf("[DEBUG] Node dispatch configuration response: %+v", dispatchResp)
			}
		}
	}

	// Set the composite ID (project:runner_id) and computed fields
	compositeId := fmt.Sprintf("%s:%s", projectName, runnerId)
	d.SetId(compositeId)
	d.Set("runner_id", runnerId)

	if response.Token != nil {
		d.Set("token", *response.Token)
	}

	if response.DownloadTk != nil {
		d.Set("download_token", *response.DownloadTk)
	}

	return ReadProjectRunner(d, meta)
}

func UpdateProjectRunner(d *schema.ResourceData, meta interface{}) error {
	clients := meta.(*RundeckClients)
	client := clients.V2
	ctx := clients.ctx

	// Parse the composite ID (project:runner_id)
	parts := strings.SplitN(d.Id(), ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid ID format, expected 'project:runner_id', got: %s", d.Id())
	}

	projectName := parts[0]
	runnerId := parts[1]

	// Debug output
	log.Printf("[DEBUG] Updating project runner - Project: %s, Runner ID: %s", projectName, runnerId)

	// Check if basic runner properties have changed (tag_names only)
	// Note: name, description, installation_type and replica_type are always included in the request
	if d.HasChange("name") || d.HasChange("description") || d.HasChange("tag_names") ||
		d.HasChange("installation_type") || d.HasChange("replica_type") {

		// Create save project runner request
		saveRequest := openapi.NewSaveProjectRunnerRequest(runnerId)

		// Always set current name value
		name := d.Get("name").(string)
		saveRequest.SetName(name)

		// Always set current description value
		description := d.Get("description").(string)
		saveRequest.SetDescription(description)

		if d.HasChange("tag_names") {
			tagNames := d.Get("tag_names").(string)
			saveRequest.SetTagNames(tagNames)
		}

		// Always set current installation_type value
		installationType := d.Get("installation_type").(string)
		if installationType == "" {
			installationType = "linux"
		}
		saveRequest.SetInstallationType(installationType)

		// Always set current replica_type value
		replicaType := d.Get("replica_type").(string)
		if replicaType == "" {
			replicaType = "manual"
		}
		saveRequest.SetReplicaType(replicaType)

		// Execute the save request
		_, resp, err := client.RunnerAPI.SaveProjectRunner(ctx, projectName, runnerId).SaveProjectRunnerRequest(*saveRequest).Execute()

		if resp.StatusCode != 200 {
			if resp != nil && resp.StatusCode == 404 {
				d.SetId("")
				return nil
			}
			return fmt.Errorf("failed to save project runner: %v", err)
		}

		log.Printf("[DEBUG] Save Project Runner API response status: %d", resp.StatusCode)
	}

	// Check if node dispatch configuration has changed
	if d.HasChange("runner_as_node_enabled") || d.HasChange("remote_node_dispatch") || d.HasChange("runner_node_filter") {

		// Create node dispatch configuration request
		nodeDispatchRequest := openapi.NewSaveProjectRunnerNodeDispatchSettingsRequest(runnerId)

		// Set all node dispatch values (not just changed ones)
		runnerAsNodeEnabled := d.Get("runner_as_node_enabled").(bool)
		nodeDispatchRequest.SetRunnerAsNodeEnabled(runnerAsNodeEnabled)

		remoteNodeDispatch := d.Get("remote_node_dispatch").(bool)
		nodeDispatchRequest.SetRemoteNodeDispatch(remoteNodeDispatch)

		runnerNodeFilter := d.Get("runner_node_filter").(string)
		if runnerNodeFilter != "" {
			nodeDispatchRequest.SetRunnerNodeFilter(runnerNodeFilter)
		}

		// Debug output
		log.Printf("[DEBUG] Updating node dispatch configuration for runner: %s in project: %s", runnerId, projectName)

		// Make the node dispatch configuration API call
		dispatchResp, dispatchHttpResp, err := client.RunnerAPI.SaveProjectRunnerNodeDispatchSettings(ctx, projectName).SaveProjectRunnerNodeDispatchSettingsRequest(*nodeDispatchRequest).Execute()

		if dispatchHttpResp.StatusCode != 200 {
			if dispatchHttpResp != nil && dispatchHttpResp.StatusCode == 404 {
				d.SetId("")
				return nil
			}
			return fmt.Errorf("failed updating node dispatch configuration for runner: %v", err)
		}

		// Debug output
		log.Printf("[DEBUG] Node dispatch update API response status: %d", dispatchHttpResp.StatusCode)
		if dispatchResp != nil {
			log.Printf("[DEBUG] Node dispatch update response: %+v", dispatchResp)
		}
	}

	// Refresh the resource data
	return ReadProjectRunner(d, meta)
}

func ReadProjectRunner(d *schema.ResourceData, meta interface{}) error {
	clients := meta.(*RundeckClients)
	client := clients.V2
	ctx := clients.ctx

	// Parse the composite ID (project:runner_id)
	parts := strings.SplitN(d.Id(), ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid ID format, expected 'project:runner_id', got: %s", d.Id())
	}

	projectName := parts[0]
	runnerId := parts[1]

	// Get runner info by ID within project context
	runnerInfo, resp, err := client.RunnerAPI.ProjectRunnerInfo(ctx, runnerId, projectName).Execute()

	// Debug output
	log.Printf("[DEBUG] ProjectRunnerInfo API call for ID: %s in project: %s", runnerId, projectName)
	if runnerInfo != nil {
		log.Printf("[DEBUG] Project Runner Info Response: %+v", runnerInfo)
	} else {
		log.Printf("[DEBUG] Project Runner Info Response: nil")
	}

	if resp.StatusCode != 200 {
		if resp != nil && resp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("failed to get project runner info: %v", err)
	}

	if runnerInfo == nil {
		d.SetId("")
		return nil
	}

	// Set the attributes
	d.Set("project_name", projectName)

	if runnerInfo.Name != nil {
		d.Set("name", *runnerInfo.Name)
	}

	if runnerInfo.Description != nil {
		d.Set("description", *runnerInfo.Description)
	}

	if runnerInfo.TagNames != nil {
		// Convert slice to comma-separated string
		tagNames := strings.Join(runnerInfo.TagNames, ",")
		d.Set("tag_names", tagNames)
	}

	if runnerInfo.Id != nil {
		d.Set("runner_id", *runnerInfo.Id)
	}

	// Set node dispatch configuration fields
	if runnerInfo.RunnerAsNodeEnabled != nil {
		d.Set("runner_as_node_enabled", *runnerInfo.RunnerAsNodeEnabled)
	}

	if runnerInfo.RemoteNodeDispatch != nil {
		d.Set("remote_node_dispatch", *runnerInfo.RemoteNodeDispatch)
	}

	if runnerInfo.RunnerNodeFilter != nil {
		d.Set("runner_node_filter", *runnerInfo.RunnerNodeFilter)
	}

	return nil
}

func DeleteProjectRunner(d *schema.ResourceData, meta interface{}) error {
	clients := meta.(*RundeckClients)
	client := clients.V2
	ctx := clients.ctx

	// Parse the composite ID (project:runner_id)
	parts := strings.SplitN(d.Id(), ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid ID format, expected 'project:runner_id', got: %s", d.Id())
	}

	projectName := parts[0]
	runnerId := parts[1]

	// Debug output
	log.Printf("[DEBUG] Deleting project runner - Project: %s, Runner ID: %s", projectName, runnerId)

	// Delete the runner from the project
	resp, err := client.RunnerAPI.DeleteProjectRunner(ctx, projectName, runnerId).Execute()
	if err != nil {
		// If the runner doesn't exist, that's okay
		if resp != nil && resp.StatusCode == 404 {
			return nil
		}
		return fmt.Errorf("failed to delete project runner: %v", err)
	}

	// Debug output
	log.Printf("[DEBUG] Delete Project Runner API response status: %d", resp.StatusCode)

	return nil
}

func resourceProjectRunnerImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	// The ID should be in the format "project:runner_id"
	parts := strings.SplitN(d.Id(), ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid ID format for import, expected 'project:runner_id', got: %s", d.Id())
	}

	projectName := parts[0]
	runnerId := parts[1]

	d.Set("project_name", projectName)
	d.Set("runner_id", runnerId)

	err := ReadProjectRunner(d, meta)
	if err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}
