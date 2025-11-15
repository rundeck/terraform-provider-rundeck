package rundeck

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type BaseKeyType int32

const (
	PRIVATE_KEY BaseKeyType = 0
	PASSWORD    BaseKeyType = 1
)

func CreateOrUpdateBaseKey(d *schema.ResourceData, meta interface{}, baseKeyType BaseKeyType) error {
	clients := meta.(*RundeckClients)
	client := clients.V1

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
		return fmt.Errorf("internal error: unknown key type")
	}

	payloadReader := io.NopCloser(strings.NewReader(payload))

	if d.Id() != "" {
		_, err = client.StorageKeyUpdate(ctx, path, payloadReader, contentType)
		if err != nil {
			return fmt.Errorf("error updating key: %v", err)
		}
	} else {
		_, err = client.StorageKeyCreate(ctx, path, payloadReader, contentType)
		if err != nil {
			return fmt.Errorf("error creating key: %v", err)
		}
	}

	d.SetId(path)

	return ReadBaseKey(d, meta)
}

func DeleteBaseKey(d *schema.ResourceData, meta interface{}) error {
	clients := meta.(*RundeckClients)
	client := clients.V1
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

	// Check if Meta or RundeckContentType is nil
	if resp.Meta == nil || resp.Meta.RundeckContentType == nil {
		return false, fmt.Errorf("response meta or content type is nil")
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
		return false, fmt.Errorf("internal error: unknown key type")
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
