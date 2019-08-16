package rundeck

import (
	"context"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/rundeck/go-rundeck/rundeck"
)

func resourceRundeckPassword() *schema.Resource {
	return &schema.Resource{
		Create: CreateOrUpdatePassword,
		Update: CreateOrUpdatePassword,
		Delete: DeletePassword,
		Exists: PasswordExists,
		Read:   ReadPassword,
		Importer: &schema.ResourceImporter{
			State: resourcePasswordImport,
		},

		Schema: map[string]*schema.Schema{
			"path": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Path to the password within the key store",
				ForceNew:    true,
			},

			"password": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The password text to store",
				Sensitive:   true,
			},
		},
	}
}

func CreateOrUpdatePassword(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*rundeck.BaseClient)

	path := d.Get("path").(string)
	password := d.Get("password").(string)

	var err error

	ctx := context.Background()

	passwordReader := ioutil.NopCloser(strings.NewReader(password))

	if d.Id() != "" {
		resp, err := client.StorageKeyUpdate(ctx, path, passwordReader, "application/x-rundeck-data-password")
		if resp.StatusCode == 409 || err != nil {
			return fmt.Errorf("Error updating or adding password: Key exists")
		}
	} else {
		resp, err := client.StorageKeyCreate(ctx, path, passwordReader, "application/x-rundeck-data-password")
		if resp.StatusCode == 409 || err != nil {
			return fmt.Errorf("Error updating or adding password: Key exists")
		}
	}

	if err != nil {
		return err
	}

	d.SetId(path)

	return ReadPassword(d, meta)
}

func DeletePassword(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*rundeck.BaseClient)
	ctx := context.Background()

	path := d.Id()

	// The only "delete" call we have is oblivious to password/key type, but
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

func ReadPassword(d *schema.ResourceData, meta interface{}) error {
	// Nothing to read for a password: existence is all we need to
	// worry about, and PasswordKeyExists took care of that.
	return nil
}

func PasswordExists(d *schema.ResourceData, meta interface{}) (bool, error) {
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

	if resp.Meta.RundeckKeyType != "" {
		// If the password isn't a password (no keytype) then as far as this resource is
		// concerned it doesn't exist. (We'll fail properly when we try to
		// create a key where one already exists.)
		return false, nil
	}

	return true, nil
}

// The below does a faux import. It will look for the secret and if it finds it exists it will make record it
func resourcePasswordImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	path := d.Id()

	exists, err := PasswordExists(d, meta)
	if !exists {
		return nil, fmt.Errorf("Unable to find password with pat %q: %q", path, err)
	}

	d.Set("path", path)
	d.Set("password", "THIS_WILL_CHANGE")
	d.SetId(path)

	return []*schema.ResourceData{d}, nil
}
