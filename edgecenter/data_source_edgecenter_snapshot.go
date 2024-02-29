package edgecenter

import (
	"context"
	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"log"
)

func dataSourceSnapshot() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceSnapshotRead,
		Description: "Represent snapshot",
		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:         schema.TypeInt,
				Computed:     true,
				Optional:     true,
				Description:  "The uuid of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			"project_name": {
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				Description:  "The name of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			"region_id": {
				Type:         schema.TypeInt,
				Computed:     true,
				Optional:     true,
				Description:  "The uuid of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			"region_name": {
				Type:         schema.TypeString,
				Computed:     true,
				Optional:     true,
				Description:  "The name of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: "The name of the snapshot. Use only with uniq name.",
			},
			"snapshot_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: "The ID of the snapshot.",
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The description of the snapshot.",
			},
			"creator_task_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The task that created this entity.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The status of the snapshot.",
			},
			"size": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The size of the snapshot, GiB.",
			},
			"volume_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: "The ID of the volume this snapshot was made from.",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The datetime when the volume was last updated.",
			},
			"task_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The active task ID this snapshot is locked by.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The datetime when the volume was created.",
			},
			"metadata": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "The metadata",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func dataSourceSnapshotRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start snapshot reading")
	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	snapshotID := d.Get("snapshot_id").(string)
	volumeID := d.Get("volume_id").(string)

	log.Printf("[DEBUG] Snapshot id = %s", snapshotID)

	if snapshotID != "" {
		snapshot, _, err := clientV2.Snapshots.Get(ctx, snapshotID)
		if err != nil {
			return diag.Errorf("cannot get snapshot with ID %s. Error: %s", snapshotID, err.Error())
		}

		setSnapshotData(d, snapshot)

		log.Println("[DEBUG] Finish snapshot reading")
		return diags
	}

	name := d.Get("name").(string)
	snapshotsOpts := &edgecloudV2.SnapshotListOptions{VolumeID: volumeID}

	if name == "" && volumeID != "" {
		return diag.Errorf("Name must be specified when VolumeID is provided")
	}
	allSnapshots, _, err := clientV2.Snapshots.List(ctx, snapshotsOpts)
	if err != nil {
		return diag.Errorf("cannot get snapshots. Error: %s", err.Error())
	}

	var foundSnapshots []*edgecloudV2.Snapshot
	for _, snapshot := range allSnapshots {
		if name == snapshot.Name {
			foundSnapshots = append(foundSnapshots, &snapshot)
		}
	}

	if len(foundSnapshots) == 0 {
		return diag.Errorf("snapshot with name %s does not exist", name)
	} else if len(foundSnapshots) > 1 {
		return diag.Errorf("multiple snapshots found with name %s. Use snapshot_id instead of name.", name)
	}

	if foundSnapshots[0].VolumeID != volumeID {
		return diag.Errorf("snapshot with name %s does not match the specified volume_id", name)
	}

	setSnapshotData(d, foundSnapshots[0])

	log.Println("[DEBUG] Finish snapshot reading")

	return diags
}

func setSnapshotData(d *schema.ResourceData, snapshot *edgecloudV2.Snapshot) {
	var diags diag.Diagnostics
	d.SetId(snapshot.ID)
	d.Set("name", snapshot.Name)
	d.Set("updated_at", snapshot.UpdatedAt)
	d.Set("created_at", snapshot.CreatedAt)
	d.Set("status", snapshot.Status)
	d.Set("creator_task_id", snapshot.CreatorTaskID)
	d.Set("size", snapshot.Size)
	d.Set("volume_id", snapshot.VolumeID)
	d.Set("description", snapshot.Description)
	d.Set("snapshot_id", snapshot.ID)

	if err := d.Set("metadata", snapshot.Metadata); err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Failed to set metadata",
			Detail:   err.Error(),
		})
	}
}
