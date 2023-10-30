package rundeck

import (
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceRundeckPrivateKey() *schema.Resource {
	return &schema.Resource{
		Create: func(rd *schema.ResourceData, i interface{}) error {
			return CreateOrUpdateBaseKey(rd, i, PRIVATE_KEY)
		},
		Update: func(rd *schema.ResourceData, i interface{}) error {
			return CreateOrUpdateBaseKey(rd, i, PRIVATE_KEY)
		},
		Delete: DeleteBaseKey,
		Exists: func(rd *schema.ResourceData, i interface{}) (bool, error) {
			return BaseKeyExists(rd, i, PRIVATE_KEY)
		},
		Read: ReadBaseKey,

		Schema: map[string]*schema.Schema{
			"path": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Path to the key within the key store",
				ForceNew:    true,
			},

			"key_material": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The private key material to store, in PEM format",
				StateFunc:   BaseKeyStateFunction,
			},
		},
	}
}
