package edgecenter

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

const (
	PortSecurityPortIDsField  = "port_ids"
	PortSecurityDisabledField = "port_security_disabled"
)

func resourcePortSecurity() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePortSecurityCreate,
		ReadContext:   resourcePortSecurityRead,
		DeleteContext: resourcePortSecurityDelete,
		Description:   "Represent port_security resource",
		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
				projectID, regionID, portID, err := ImportStringParser(d.Id())
				if err != nil {
					return nil, err
				}
				d.Set(ProjectIDField, projectID)
				d.Set(RegionIDField, regionID)
				d.SetId(portID)

				return []*schema.ResourceData{d}, nil
			},
		},

		Schema: map[string]*schema.Schema{
			ProjectIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				Description:  "The uuid of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{ProjectIDField, ProjectNameField},
			},
			ProjectNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Description:  "The name of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{ProjectIDField, ProjectNameField},
			},
			RegionIDField: {
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				Description:  "The uuid of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{RegionIDField, RegionNameField},
			},
			RegionNameField: {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Description:  "The name of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{RegionIDField, RegionNameField},
			},

			InstanceIDField: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "ID of the instance to which the ports are connected.",
			},

			PortSecurityDisabledField: {
				Type:        schema.TypeBool,
				Description: "Is the port_security feature disabled.",
				Default:     false,
				Optional:    true,
			},
			PortSecurityPortIDsField: {
				Type:        schema.TypeList,
				Description: "List of security group IDs.",
				Required:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			SecurityGroupsField: {
				Type:        schema.TypeList,
				Description: "The ID of the port.",
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourcePortSecurityCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start port_security creating")

	clientV2, err := InitCloudClient(ctx, d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	diags := validatePortSecAttrs(d)
	if diags.HasError() {
		return diags
	}
	portsIDsRaw := d.Get(PortSecurityPortIDsField).([]interface{})
	portSecurityDisabled := d.Get(PortSecurityDisabledField).(bool)
	if portSecurityDisabled {
		for _, portIDRaw := range portsIDsRaw {
			portID := portIDRaw.(string)
			clientV2.Ports.DisablePortSecurity(ctx, portID)
		}
		return diags
	}

	for _, portIDRaw := range portsIDsRaw {
		portID := portIDRaw.(string)
		clientV2.Ports.DisablePortSecurity(ctx, portID)
	}

	portSGNames := edgecloudV2.PortsSecurityGroupNames{
		SecurityGroupNames: []string{sgInfo.Name},
		PortID:             portID,
	}
	sgOpts := edgecloudV2.AssignSecurityGroupRequest{PortsSecurityGroupNames: []edgecloudV2.PortsSecurityGroupNames{portSGNames}}

	log.Printf("[DEBUG] attach security group opts: %+v", sgOpts)

	if _, err := clientV2.Instances.SecurityGroupAssign(ctx, instanceID, &sgOpts); err != nil {
		return diag.Errorf("cannot attach security group. Error: %w", err)
	}

	d.SetId(serverGroup.ID)
	resourcePortSecurityRead(ctx, d, m)
	log.Println("[DEBUG] Finish ServerGroup creating")

	return diags
}

func resourcePortSecurityRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start ServerGroup reading")
	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID
	d.Set("project_id", projectID)
	d.Set("region_id", regionID)

	serverGroup, _, err := clientV2.ServerGroups.Get(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("name", serverGroup.Name)
	d.Set("policy", serverGroup.Policy)

	instances := make([]map[string]string, len(serverGroup.Instances))
	for i, instance := range serverGroup.Instances {
		rawInstance := make(map[string]string)
		rawInstance["instance_id"] = instance.InstanceID
		rawInstance["instance_name"] = instance.InstanceName
		instances[i] = rawInstance
	}
	if err := d.Set("instances", instances); err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish ServerGroup reading")

	return diags
}

func resourcePortSecurityDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start ServerGroup deleting")
	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient

	regionID, projectID, err := GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	clientV2.Region = regionID
	clientV2.Project = projectID

	_, err = clientV2.ServerGroups.Delete(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	log.Println("[DEBUG] Finish ServerGroup deleting")

	return diags
}
