package rundeck

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceRundeckPassword() *schema.Resource {
	return &schema.Resource{
		Create: func(rd *schema.ResourceData, i interface{}) error {
			return CreateOrUpdateBaseKey(rd, i, PASSWORD)
		},
		Update: func(rd *schema.ResourceData, i interface{}) error {
			return CreateOrUpdateBaseKey(rd, i, PASSWORD)
		},
		Delete: DeleteBaseKey,
		Exists: func(rd *schema.ResourceData, i interface{}) (bool, error) {
			return BaseKeyExists(rd, i, PASSWORD)
		},
		Read: ReadBaseKey,

		Schema: map[string]*schema.Schema{
			"path": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Path to the key within the key store",
				ForceNew:    true,
			},

			"password": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The password to store",
				StateFunc:   BaseKeyStateFunction,
			},
		},
	}
}
