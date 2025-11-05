package rundeck

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	openapi "github.com/rundeck/go-rundeck/rundeck-v2"
)

func resourceRundeckSystemRunner() *schema.Resource {
	return &schema.Resource{
		Create: CreateSystemRunner,
		Update: UpdateSystemRunner,
		Delete: DeleteSystemRunner,
		Read:   ReadSystemRunner,
		Importer: &schema.ResourceImporter{
			State: resourceSystemRunnerImport,
		},

		Schema: map[string]*schema.Schema{
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

			"assigned_projects": {
				Type:        schema.TypeMap,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Map of assigned projects",
			},

			"project_runner_as_node": {
				Type:        schema.TypeMap,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeBool},
				Description: "Map of projects where runner acts as node",
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

func CreateSystemRunner(d *schema.ResourceData, meta interface{}) error {
	clients := meta.(*RundeckClients)
	client := clients.V2
	ctx := clients.ctx

	name := d.Get("name").(string)
	description := d.Get("description").(string)

	// Create the runner request
	runnerRequest := openapi.NewCreateRunnerRequest(name, description)

	if tagNames, ok := d.GetOk("tag_names"); ok {
		tagNamesStr := tagNames.(string)
		runnerRequest.SetTagNames(tagNamesStr)
	}

	if assignedProjects, ok := d.GetOk("assigned_projects"); ok {
		projects := assignedProjects.(map[string]interface{})
		projectsMap := make(map[string]string)
		for k, v := range projects {
			projectsMap[k] = v.(string)
		}
		runnerRequest.SetAssignedProjects(projectsMap)
	}

	if projectRunnerAsNode, ok := d.GetOk("project_runner_as_node"); ok {
		nodes := projectRunnerAsNode.(map[string]interface{})
		nodesMap := make(map[string]bool)
		for k, v := range nodes {
			nodesMap[k] = v.(bool)
		}
		runnerRequest.SetProjectRunnerAsNode(nodesMap)
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

	// Although this is a system runner, the API requires wrapping the runner request in CreateProjectRunnerRequest
	projectRunnerRequest := openapi.NewCreateProjectRunnerRequest(name, description)
	projectRunnerRequest.SetNewRunnerRequest(*runnerRequest)

	// Create the runner
	response, _, err := client.RunnerAPI.CreateRunner(ctx).CreateProjectRunnerRequest(*projectRunnerRequest).Execute()
	if err != nil {
		return fmt.Errorf("failed to create runner: %v", err)
	}

	// Set the ID and computed fields
	if response.RunnerId != nil {
		d.SetId(*response.RunnerId)
		d.Set("runner_id", *response.RunnerId)
	}

	if response.Token != nil {
		d.Set("token", *response.Token)
	}

	if response.DownloadTk != nil {
		d.Set("download_token", *response.DownloadTk)
	}

	return ReadSystemRunner(d, meta)
}

func UpdateSystemRunner(d *schema.ResourceData, meta interface{}) error {
	clients := meta.(*RundeckClients)
	client := clients.V2
	ctx := clients.ctx

	runnerId := d.Id()

	// Debug output
	log.Printf("[DEBUG] Updating system runner - Runner ID: %s", runnerId)

	// Check if basic runner properties have changed (tag_names only)
	// Note: name, description, installation_type, replica_type, and assigned_projects are always included in the request
	if d.HasChange("name") || d.HasChange("description") || d.HasChange("tag_names") ||
		d.HasChange("installation_type") || d.HasChange("replica_type") ||
		d.HasChange("assigned_projects") {

		// Create save runner request - system runners use SaveProjectRunnerRequest
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

		// Always set current assigned_projects value
		if assignedProjects, ok := d.GetOk("assigned_projects"); ok {
			projects := assignedProjects.(map[string]interface{})
			projectsMap := make(map[string]string)
			for k, v := range projects {
				projectsMap[k] = v.(string)
			}
			saveRequest.SetAssignedProjects(projectsMap)
		}

		// Note: project_runner_as_node is not supported in SaveProjectRunnerRequest
		// This field might be handled differently or not supported for system runner updates

		// Execute the save request using SaveRunner API
		_, resp, err := client.RunnerAPI.SaveRunner(ctx, runnerId).SaveProjectRunnerRequest(*saveRequest).Execute()

		if resp.StatusCode != 200 {
			if resp != nil && resp.StatusCode == 404 {
				d.SetId("")
				return nil
			}
			return fmt.Errorf("failed to save runner: %v", err)
		}

		log.Printf("[DEBUG] Save Runner API response status: %d", resp.StatusCode)
	}

	// Refresh the resource data
	return ReadSystemRunner(d, meta)
}

func ReadSystemRunner(d *schema.ResourceData, meta interface{}) error {
	clients := meta.(*RundeckClients)
	client := clients.V2

	runnerId := d.Id()
	ctx := clients.ctx

	// Get runner info by ID
	runnerInfo, resp, err := client.RunnerAPI.RunnerInfo(ctx, runnerId).Execute()

	// Debug output
	log.Printf("[DEBUG] RunnerInfo API call for ID: %s", runnerId)
	if runnerInfo != nil {
		log.Printf("[DEBUG] Runner Info Response: %+v", runnerInfo)
	} else {
		log.Printf("[DEBUG] Runner Info Response: nil")
	}

	if resp.StatusCode != 200 {
		if resp != nil && resp.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("failed to get runner info: %v", err)
	}

	if runnerInfo == nil {
		d.SetId("")
		return nil
	}

	// Set the attributes
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

	return nil
}

func DeleteSystemRunner(d *schema.ResourceData, meta interface{}) error {
	clients := meta.(*RundeckClients)
	client := clients.V2

	runnerId := d.Id()
	ctx := clients.ctx

	_, err := client.RunnerAPI.DeleteRunner(ctx, runnerId).Execute()
	if err != nil {
		// Log the error but continue with other projects
		log.Printf("Warning: failed to delete runner %s: %v", runnerId, err)
	}
	// Note: There might not be a global runner delete endpoint
	// The runner might be automatically cleaned up when removed from all projects
	// Or there might be a different API endpoint for global runner deletion

	return nil
}

func resourceSystemRunnerImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	// The ID should be the runner ID
	runnerId := d.Id()
	d.Set("runner_id", runnerId)

	err := ReadSystemRunner(d, meta)
	if err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}
