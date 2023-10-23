package rundeck

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"

	"github.com/rundeck/go-rundeck/rundeck"
)

type BaseKeyType int32

const (
	PRIVATE_KEY BaseKeyType = 0
	PASSWORD    BaseKeyType = 1
)

func CreateOrUpdateBaseKey(d *schema.ResourceData, meta interface{}, baseKeyType BaseKeyType) error {
	client := meta.(*rundeck.BaseClient)

	var payload string
	path := d.Get("path").(string)

	var err error

	ctx := context.Background()

	var contentType string
	switch baseKeyType {
	case PRIVATE_KEY:
		contentType = "application/octet-stream"
		payload = d.Get("key_material").(string)
	case PASSWORD:
		contentType = "application/x-rundeck-data-password"
		payload = d.Get("password").(string)
	default:
		return fmt.Errorf("Internal error. Unknown key type")
	}

	payloadReader := io.NopCloser(strings.NewReader(payload))

	if d.Id() != "" {
		resp, err := client.StorageKeyUpdate(ctx, path, payloadReader, contentType)
		if resp.StatusCode == 409 || err != nil {
			return fmt.Errorf("Error updating or adding key: Key exists")
		}
	} else {
		resp, err := client.StorageKeyCreate(ctx, path, payloadReader, contentType)
		if resp.StatusCode == 409 || err != nil {
			return fmt.Errorf("Error updating or adding key: Key exists")
		}
	}

	if err != nil {
		return err
	}

	d.SetId(path)

	return ReadBaseKey(d, meta)
}

func DeleteBaseKey(d *schema.ResourceData, meta interface{}) error {
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

func ReadBaseKey(d *schema.ResourceData, meta interface{}) error {
	// Nothing to read for a private key: existence is all we need to
	// worry about, and PrivateKeyExists took care of that.
	return nil
}

func BaseKeyExists(d *schema.ResourceData, meta interface{}, baseKeyType BaseKeyType) (bool, error) {
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

	// If the resource is not password or private key as far as this resource is
	// concerned it doesn't exist. (We'll fail properly when we try to
	// create a key where one already exists.)
	//  Content-type are:
	//  application/octet-stream specifies a private key
	//  application/pgp-keys specifies a public key
	//  application/x-rundeck-data-password specifies a password
	switch baseKeyType {
	case PRIVATE_KEY:
		if *resp.Meta.RundeckContentType != "application/octet-stream" {
			return false, nil
		}
	case PASSWORD:
		if *resp.Meta.RundeckContentType != "application/x-rundeck-data-password" {
			return false, nil
		}
	default:
		return false, fmt.Errorf("Internal error. Unknown key type")
	}

	return true, nil
}

func BaseKeyStateFunction(val interface{}) string {
	switch v := val.(type) {
	case string:
		hash := sha1.Sum([]byte(v))
		return hex.EncodeToString(hash[:])
	default:
		return ""
	}
}
