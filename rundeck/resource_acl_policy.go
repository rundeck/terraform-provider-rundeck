package rundeck

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/rundeck/go-rundeck/rundeck"
)

func resourceRundeckAclPolicy() *schema.Resource {
	return &schema.Resource{
		Create: CreateAclPolicy,
		Update: UpdateAclPolicy,
		Read:   ReadAclPolicy,
		Delete: AclPolicyDelete,
		Exists: AclPolicyExists,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Unique name for the ACL policy",
				ForceNew:    true,
			},
			"policy": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "YAML formatted ACL Policy string",
				ForceNew:    false,
			},
		},
	}
}

func CreateAclPolicy(d *schema.ResourceData, meta interface{}) error {
	clients := meta.(*RundeckClients)
	client := clients.V1

	name := d.Get("name").(string)
	policy := d.Get("policy").(string)

	ctx := context.Background()

	req := &rundeck.SystemACLPolicyCreateRequest{
		Contents: &policy,
	}

	resp, err := client.SystemACLPolicyCreate(ctx, name, req)
	if err != nil {
		return err
	}

	if resp.StatusCode == 409 || resp.StatusCode == 400 {
		return fmt.Errorf("Error creating ACL policy: (%v)", resp.Value)
	}

	d.SetId(name)
	val := d.Set("id", name)
	if val != nil {
	}

	return nil
}

func UpdateAclPolicy(d *schema.ResourceData, meta interface{}) error {
	clients := meta.(*RundeckClients)
	client := clients.V1

	name := d.Get("name").(string)
	policy := d.Get("policy").(string)

	ctx := context.Background()

	req := &rundeck.SystemACLPolicyUpdateRequest{
		Contents: &policy,
	}

	_, err := client.SystemACLPolicyUpdate(ctx, name, req)

	if err != nil {
		return err
	}

	return nil
}

func ReadAclPolicy(d *schema.ResourceData, meta interface{}) error {
	clients := meta.(*RundeckClients)
	client := clients.V1
	ctx := context.Background()

	name := d.Id()
	resp, err := client.SystemACLPolicyGet(ctx, name)

	if err != nil {
		return err
	}

	if resp.StatusCode == 404 {
		return fmt.Errorf("ACL policy not found: (%s)", name)
	}

	val := d.Set("policy", *resp.Contents)
	if val != nil {
	}

	return nil
}

func AclPolicyExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	clients := meta.(*RundeckClients)
	client := clients.V1
	ctx := context.Background()

	name := d.Id()

	resp, err := client.SystemACLPolicyGet(ctx, name)
	if err != nil {
		return false, err
	}
	if resp.StatusCode == 200 {
		return true, nil
	}
	if resp.StatusCode == 404 {
		return false, nil
	}

	return false, err
}

func AclPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	clients := meta.(*RundeckClients)
	client := clients.V1
	ctx := context.Background()

	name := d.Id()

	_, err := client.SystemACLPolicyDelete(ctx, name)

	return err
}
