package rundeck

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"

	"github.com/rundeck/go-rundeck/rundeck"
)

func resourceRundeckPublicKey() *schema.Resource {
	return &schema.Resource{
		Create: CreatePublicKey,
		Update: UpdatePublicKey,
		Delete: DeletePublicKey,
		Exists: PublicKeyExists,
		Read:   ReadPublicKey,

		Schema: map[string]*schema.Schema{
			"path": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Path to the key within the key store",
				ForceNew:    true,
			},

			"key_material": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The public key data to store, in the usual OpenSSH public key file format",
			},

			"url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "URL at which the key content can be retrieved",
			},

			"delete": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "True if the key should be deleted when the resource is deleted. Defaults to true if key_material is provided in the configuration.",
			},
		},
	}
}

func CreatePublicKey(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*rundeck.BaseClient)
	ctx := context.Background()

	path := d.Get("path").(string)
	keyMaterial := d.Get("key_material").(string)

	keyMaterialReader := ioutil.NopCloser(strings.NewReader(keyMaterial))

	if keyMaterial != "" {
		resp, err := client.StorageKeyCreate(ctx, path, keyMaterialReader, "application/pgp-keys")
		if resp.StatusCode == 409 {
			err = fmt.Errorf("conflict creating key at : %s", path)
		}
		if err != nil {
			return err
		}
		val := d.Set("delete", true)
		if val != nil {
			fmt.Printf("[Error]")
		}

	}

	d.SetId(path)

	return ReadPublicKey(d, meta)
}

func UpdatePublicKey(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*rundeck.BaseClient)
	ctx := context.Background()

	if d.HasChange("key_material") {
		path := d.Get("path").(string)
		keyMaterial := d.Get("key_material").(string)

		keyMaterialReader := ioutil.NopCloser(strings.NewReader(keyMaterial))

		resp, err := client.StorageKeyUpdate(ctx, path, keyMaterialReader, "application/pgp-keys")
		if resp.StatusCode == 409 || err != nil {
			return fmt.Errorf("Error updating or adding key: Key exists")
		}
		if err != nil {
			return err
		}
	}

	return ReadPublicKey(d, meta)
}

func DeletePublicKey(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*rundeck.BaseClient)
	ctx := context.Background()

	path := d.Id()

	// Since this resource can be used both to create and to read existing
	// public keys, we'll only actually delete the key if we remember that
	// we created the key in the first place, or if the user explicitly
	// opted in to have an existing key deleted.
	if d.Get("delete").(bool) {
		// The only "delete" call we have is oblivious to key type, but
		// that's okay since our Exists implementation makes sure that we
		// won't try to delete a key of the wrong type since we'll pretend
		// that it's already been deleted.
		_, err := client.StorageKeyDelete(ctx, path)
		if err != nil {
			return err
		}
	}

	d.SetId("")
	return nil
}

func ReadPublicKey(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*rundeck.BaseClient)
	ctx := context.Background()

	path := d.Id()

	key, err := client.StorageKeyGetMetadata(ctx, path)
	if key.StatusCode == 404 {
		err = fmt.Errorf("key not found at: %s", path)
	}

	if err != nil {
		return err
	}

	resp, err := client.StorageKeyGetMaterial(ctx, path)
	if err != nil {
		return err
	}

	keyMaterial, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	val1 := d.Set("key_material", string(keyMaterial))
	if val1 != nil {
		fmt.Printf("[Error]")
	}

	val2 := d.Set("url", *key.URL)
	if val2 != nil {
		fmt.Printf("[Error]")
	}

	return nil
}

func PublicKeyExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*rundeck.BaseClient)
	ctx := context.Background()

	path := d.Id()

	resp, err := client.StorageKeyGetMetadata(ctx, path)
	if resp.StatusCode == 404 || err != nil {
		return false, err
	}

	if resp.Meta.RundeckKeyType != rundeck.Public {
		// If the key type isn't public then as far as this resource is
		// concerned it doesn't exist. (We'll fail properly when we try to
		// create a key where one already exists.)
		return false, nil
	}

	return true, nil
}
