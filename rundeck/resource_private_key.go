package rundeck

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"

	"github.com/rundeck/go-rundeck/rundeck"
)

func resourceRundeckPrivateKey() *schema.Resource {
	return &schema.Resource{
		Create: CreateOrUpdatePrivateKey,
		Update: CreateOrUpdatePrivateKey,
		Delete: DeletePrivateKey,
		Exists: PrivateKeyExists,
		Read:   ReadPrivateKey,

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
				StateFunc: func(val interface{}) string {
					switch v := val.(type) {
					case string:
						hash := sha1.Sum([]byte(v))
						return hex.EncodeToString(hash[:])
					default:
						return ""
					}
				},
			},
		},
	}
}

func CreateOrUpdatePrivateKey(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*rundeck.BaseClient)

	path := d.Get("path").(string)
	keyMaterial := d.Get("key_material").(string)

	var err error

	ctx := context.Background()

	keyMaterialReader := ioutil.NopCloser(strings.NewReader(keyMaterial))

	if d.Id() != "" {
		resp, err := client.StorageKeyUpdate(ctx, path, keyMaterialReader, "application/octect-stream")
		if resp.StatusCode == 409 || err != nil {
			return fmt.Errorf("Error updating or adding key: Key exists")
		}
	} else {
		resp, err := client.StorageKeyCreate(ctx, path, keyMaterialReader, "application/octet-stream")
		if resp.StatusCode == 409 || err != nil {
			return fmt.Errorf("Error updating or adding key: Key exists")
		}
	}

	if err != nil {
		return err
	}

	d.SetId(path)

	return ReadPrivateKey(d, meta)
}

func DeletePrivateKey(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*rundeck.BaseClient)
	ctx := context.Background()

	path := d.Id()

	// The only "delete" call we have is oblivious to key type, but
	// that's okay since our Exists implementation makes sure that we
	// won't try to delete a key of the wrong type since we'll pretend
	// that it's already been deleted.
	_, err := client.StorageKeyDelete(ctx, path)
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

func ReadPrivateKey(d *schema.ResourceData, meta interface{}) error {
	// Nothing to read for a private key: existence is all we need to
	// worry about, and PrivateKeyExists took care of that.
	return nil
}

func PrivateKeyExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*rundeck.BaseClient)
	ctx := context.Background()

	path := d.Id()

	resp, err := client.StorageKeyGetMetadata(ctx, path)
	if err != nil {
		if resp.StatusCode != 404 {
			err = nil
		}
		return false, err
	}

	if resp.Meta.RundeckKeyType != rundeck.Private {
		// If the key type isn't public then as far as this resource is
		// concerned it doesn't exist. (We'll fail properly when we try to
		// create a key where one already exists.)
		return false, nil
	}

	return true, nil
}
