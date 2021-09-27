package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/xray"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/aws/internal/service/xray/waiter"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
)

func ResourceEncryptionConfig() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsXrayEncryptionConfigPut,
		Read:   resourceEncryptionConfigRead,
		Update: resourceAwsXrayEncryptionConfigPut,
		Delete: schema.Noop,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"key_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateArn,
			},
			"type": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice([]string{
					xray.EncryptionTypeKms,
					xray.EncryptionTypeNone,
				}, false),
			},
		},
	}
}

func resourceAwsXrayEncryptionConfigPut(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).XRayConn

	input := &xray.PutEncryptionConfigInput{
		Type: aws.String(d.Get("type").(string)),
	}

	if v, ok := d.GetOk("key_id"); ok {
		input.KeyId = aws.String(v.(string))
	}

	_, err := conn.PutEncryptionConfig(input)
	if err != nil {
		return fmt.Errorf("error creating XRay Encryption Config: %w", err)
	}

	d.SetId(meta.(*conns.AWSClient).Region)

	if _, err := waiter.EncryptionConfigAvailable(conn); err != nil {
		return fmt.Errorf("error waiting for Xray Encryption Config (%s) to Available: %w", d.Id(), err)
	}

	return resourceEncryptionConfigRead(d, meta)
}

func resourceEncryptionConfigRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).XRayConn

	config, err := conn.GetEncryptionConfig(&xray.GetEncryptionConfigInput{})

	if err != nil {
		return fmt.Errorf("error reading XRay Encryption Config: %w", err)
	}

	d.Set("key_id", config.EncryptionConfig.KeyId)
	d.Set("type", config.EncryptionConfig.Type)

	return nil
}
