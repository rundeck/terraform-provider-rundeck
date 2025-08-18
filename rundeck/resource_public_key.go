package rundeck

import (
	"context"
	"fmt"
	"io"
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
	clients := meta.(*RundeckClients)
	client := clients.V1
	ctx := context.Background()

	path := d.Get("path").(string)
	keyMaterial := d.Get("key_material").(string)

	keyMaterialReader := io.NopCloser(strings.NewReader(keyMaterial))

	if keyMaterial != "" {
		_, err := client.StorageKeyCreate(ctx, path, keyMaterialReader, "application/pgp-keys")
		if err != nil {
			return fmt.Errorf("error creating key at %s: %v", path, err)
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
	clients := meta.(*RundeckClients)
	client := clients.V1
	ctx := context.Background()

	if d.HasChange("key_material") {
		path := d.Get("path").(string)
		keyMaterial := d.Get("key_material").(string)

		keyMaterialReader := io.NopCloser(strings.NewReader(keyMaterial))

		_, err := client.StorageKeyUpdate(ctx, path, keyMaterialReader, "application/pgp-keys")
		if err != nil {
			return fmt.Errorf("error updating key: %v", err)
		}
	}

	return ReadPublicKey(d, meta)
}

func DeletePublicKey(d *schema.ResourceData, meta interface{}) error {
	clients := meta.(*RundeckClients)
	client := clients.V1
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
	clients := meta.(*RundeckClients)
	client := clients.V1
	ctx := context.Background()

	path := d.Id()

	key, err := client.StorageKeyGetMetadata(ctx, path)
	if err != nil {
		return err
	}

	if key.StatusCode == 404 {
		return fmt.Errorf("key not found at: %s", path)
	}

	if key.Meta == nil || key.URL == nil {
		return fmt.Errorf("invalid response: meta or URL is nil")
	}

	resp, err := client.StorageKeyGetMaterial(ctx, path)
	if err != nil {
		return err
	}

	keyMaterial, err := io.ReadAll(resp.Body)
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
	clients := meta.(*RundeckClients)
	client := clients.V1
	ctx := context.Background()

	path := d.Id()

	resp, err := client.StorageKeyGetMetadata(ctx, path)
	if err != nil {
		return false, err
	}

	if resp.StatusCode == 404 {
		return false, nil
	}

	if resp.Meta == nil {
		return false, fmt.Errorf("response meta is nil")
	}

	if resp.Meta.RundeckKeyType != rundeck.Public {
		// If the key type isn't public then as far as this resource is
		// concerned it doesn't exist. (We'll fail properly when we try to
		// create a key where one already exists.)
		return false, nil
	}

	return true, nil
}
