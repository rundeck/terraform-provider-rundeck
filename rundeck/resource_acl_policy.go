package rundeck

import (
	"context"

	"github.com/hashicorp/terraform/helper/schema"
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
	client := meta.(*rundeck.BaseClient)

	name := d.Get("name").(string)
	policy := d.Get("policy").(string)

	ctx := context.Background()

	req := &rundeck.SystemACLPolicyCreateRequest{
		Contents: &policy,
	}

	_, err := client.SystemACLPolicyCreate(ctx, name, req)

	if err != nil {
		return err
	}

	d.SetId(name)
	d.Set("id", name)

	return nil
}

func UpdateAclPolicy(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*rundeck.BaseClient)

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
	client := meta.(*rundeck.BaseClient)
	ctx := context.Background()

	name := d.Id()
	resp, err := client.SystemACLPolicyGet(ctx, name)

	if err != nil {
		return err
	}

	d.Set("policy", *resp.Contents)

	return nil
}

func AclPolicyExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*rundeck.BaseClient)
	ctx := context.Background()

	name := d.Id()

	resp, err := client.SystemACLPolicyGet(ctx, name)

	if err == nil && resp.StatusCode == 200 {
		return true, nil
	} else if err != nil && resp.StatusCode == 404 {
		return false, nil
	}

	return false, err
}

func AclPolicyDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*rundeck.BaseClient)
	ctx := context.Background()

	name := d.Id()

	_, err := client.SystemACLPolicyDelete(ctx, name)

	return err
}
