package memorydb

import (
	"context"
	"log"
	"regexp"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/memorydb"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/create"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/internal/verify"
)

func ResourceSnapshot() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceSnapshotCreate,
		ReadContext:   resourceSnapshotRead,
		UpdateContext: resourceSnapshotUpdate,
		DeleteContext: resourceSnapshotDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(snapshotAvailableTimeout),
			Delete: schema.DefaultTimeout(snapshotDeletedTimeout),
		},

		CustomizeDiff: verify.SetTagsDiff,

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"cluster_configuration": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"description": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"engine_version": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"maintenance_window": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"node_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"num_shards": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"parameter_group_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"port": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"snapshot_retention_limit": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"snapshot_window": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"subnet_group_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"topic_arn": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"vpc_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"cluster_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"kms_key_arn": {
				// The API will accept an ID, but return the ARN on every read.
				// For the sake of consistency, force everyone to use ARN-s.
				// To prevent confusion, the attribute is suffixed _arn rather
				// than the _id implied by the API.
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: verify.ValidARN,
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
				ValidateFunc: validation.All(
					validation.StringLenBetween(1, 255),
					validation.StringDoesNotMatch(
						regexp.MustCompile(`[-][-]`),
						"The name may not contain two consecutive hyphens."),
					validation.StringMatch(
						// Similar to ElastiCache, MemoryDB normalises names to lowercase.
						regexp.MustCompile(`^[a-z0-9-]*[a-z0-9]$`),
						"Only lowercase alphanumeric characters and hyphens allowed. The name may not end with a hyphen."),
				),
			},
			"name_prefix": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name"},
				ValidateFunc: validation.All(
					validation.StringLenBetween(1, 255-resource.UniqueIDSuffixLength),
					validation.StringDoesNotMatch(
						regexp.MustCompile(`[-][-]`),
						"The name may not contain two consecutive hyphens."),
					validation.StringMatch(
						// Similar to ElastiCache, MemoryDB normalises names to lowercase.
						regexp.MustCompile(`^[a-z0-9-]+$`),
						"Only lowercase alphanumeric characters and hyphens allowed."),
				),
			},
			"source": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags":     tftags.TagsSchema(),
			"tags_all": tftags.TagsSchemaComputed(),
		},
	}
}

func resourceSnapshotCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).MemoryDBConn
	defaultTagsConfig := meta.(*conns.AWSClient).DefaultTagsConfig
	tags := defaultTagsConfig.MergeTags(tftags.New(d.Get("tags").(map[string]interface{})))

	name := create.Name(d.Get("name").(string), d.Get("name_prefix").(string))
	input := &memorydb.CreateSnapshotInput{
		ClusterName:  aws.String(d.Get("cluster_name").(string)),
		SnapshotName: aws.String(name),
		Tags:         Tags(tags.IgnoreAWS()),
	}

	if v, ok := d.GetOk("kms_key_arn"); ok {
		input.KmsKeyId = aws.String(v.(string))
	}

	log.Printf("[DEBUG] Creating MemoryDB Snapshot: %s", input)
	_, err := conn.CreateSnapshotWithContext(ctx, input)

	if err != nil {
		return diag.Errorf("error creating MemoryDB Snapshot (%s): %s", name, err)
	}

	if err := waitSnapshotAvailable(ctx, conn, name, d.Timeout(schema.TimeoutCreate)); err != nil {
		return diag.Errorf("error waiting for MemoryDB Snapshot (%s) to be created: %s", name, err)
	}

	d.SetId(name)

	return resourceSnapshotRead(ctx, d, meta)
}

func resourceSnapshotUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).MemoryDBConn

	if d.HasChange("tags_all") {
		o, n := d.GetChange("tags_all")

		if err := UpdateTags(conn, d.Get("arn").(string), o, n); err != nil {
			return diag.Errorf("error updating MemoryDB Snapshot (%s) tags: %s", d.Id(), err)
		}
	}

	return resourceSnapshotRead(ctx, d, meta)
}

func resourceSnapshotRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).MemoryDBConn
	defaultTagsConfig := meta.(*conns.AWSClient).DefaultTagsConfig
	ignoreTagsConfig := meta.(*conns.AWSClient).IgnoreTagsConfig

	snapshot, err := FindSnapshotByName(ctx, conn, d.Id())

	if !d.IsNewResource() && tfresource.NotFound(err) {
		log.Printf("[WARN] MemoryDB Snapshot (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return diag.Errorf("error reading MemoryDB Snapshot (%s): %s", d.Id(), err)
	}

	d.Set("arn", snapshot.ARN)
	if err := d.Set("cluster_configuration", flattenClusterConfiguration(snapshot.ClusterConfiguration)); err != nil {
		return diag.Errorf("failed to set cluster_configuration for MemoryDB Snapshot (%s): %s", d.Id(), err)
	}
	d.Set("cluster_name", snapshot.ClusterConfiguration.Name)
	d.Set("kms_key_arn", snapshot.KmsKeyId)
	d.Set("name", snapshot.Name)
	d.Set("name_prefix", create.NamePrefixFromName(aws.StringValue(snapshot.Name)))
	d.Set("source", snapshot.Source)

	tags, err := ListTags(conn, d.Get("arn").(string))

	if err != nil {
		return diag.Errorf("error listing tags for MemoryDB Snapshot (%s): %s", d.Id(), err)
	}

	tags = tags.IgnoreAWS().IgnoreConfig(ignoreTagsConfig)

	//lintignore:AWSR002
	if err := d.Set("tags", tags.RemoveDefaultConfig(defaultTagsConfig).Map()); err != nil {
		return diag.Errorf("error setting tags for MemoryDB Snapshot (%s): %s", d.Id(), err)
	}

	if err := d.Set("tags_all", tags.Map()); err != nil {
		return diag.Errorf("error setting tags_all for MemoryDB Snapshot (%s): %s", d.Id(), err)
	}

	return nil
}

func resourceSnapshotDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).MemoryDBConn

	log.Printf("[DEBUG] Deleting MemoryDB Snapshot: (%s)", d.Id())
	_, err := conn.DeleteSnapshotWithContext(ctx, &memorydb.DeleteSnapshotInput{
		SnapshotName: aws.String(d.Id()),
	})

	if tfawserr.ErrCodeEquals(err, memorydb.ErrCodeSnapshotNotFoundFault) {
		return nil
	}

	if err != nil {
		return diag.Errorf("error deleting MemoryDB Snapshot (%s): %s", d.Id(), err)
	}

	if err := waitSnapshotDeleted(ctx, conn, d.Id(), d.Timeout(schema.TimeoutDelete)); err != nil {
		return diag.Errorf("error waiting for MemoryDB Snapshot (%s) to be deleted: %s", d.Id(), err)
	}

	return nil
}

func flattenClusterConfiguration(v *memorydb.ClusterConfiguration) []interface{} {
	if v == nil {
		return []interface{}{}
	}

	m := map[string]interface{}{
		"description":              aws.StringValue(v.Description),
		"engine_version":           aws.StringValue(v.EngineVersion),
		"maintenance_window":       aws.StringValue(v.MaintenanceWindow),
		"name":                     aws.StringValue(v.Name),
		"node_type":                aws.StringValue(v.NodeType),
		"num_shards":               aws.Int64Value(v.NumShards),
		"parameter_group_name":     aws.StringValue(v.ParameterGroupName),
		"port":                     aws.Int64Value(v.Port),
		"snapshot_retention_limit": aws.Int64Value(v.SnapshotRetentionLimit),
		"snapshot_window":          aws.StringValue(v.SnapshotWindow),
		"subnet_group_name":        aws.StringValue(v.SubnetGroupName),
		"topic_arn":                aws.StringValue(v.TopicArn),
		"vpc_id":                   aws.StringValue(v.VpcId),
	}

	return []interface{}{m}
}
