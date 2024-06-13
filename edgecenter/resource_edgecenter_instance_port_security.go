package edgecenter

import (
	"context"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

const (
	PortSecurityPortIDField   = "port_id"
	PortSecurityDisabledField = "port_security_disabled"
)

func resourceInstancePortSecurity() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceInstancePortSecurityCreate,
		ReadContext:   resourceInstancePortSecurityRead,
		DeleteContext: resourceInstancePortSecurityDelete,
		Description:   "Represent instance_port_security resource",
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
				Description: "ID of the instance to which the port is connected.",
			},

			PortSecurityDisabledField: {
				Type:        schema.TypeBool,
				Description: "Is the port_security feature disabled.",
				Default:     false,
				Optional:    true,
			},
			PortIDField: {
				Type:        schema.TypeList,
				Description: "ID of the port.",
				Required:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			SecurityGroupsField: {
				Type:        schema.TypeList,
				Description: "List of security groups IDs.",
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func resourceInstancePortSecurityCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start port_security creating")

	clientV2, err := InitCloudClient(ctx, d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	diags := validatePortSecAttrs(d)
	if diags.HasError() {
		return diags
	}
	portID := d.Get(PortIDField).(string)
	instanceID := d.Get(InstanceIDField).(string)

	instanceIfacePort, err := utilV2.InstanceNetworkInterfaceByID(ctx, clientV2, instanceID, portID)
	if err != nil {
		return diag.FromErr(err)
	}
	portSecurityDisabled := d.Get(PortSecurityDisabledField).(bool)

	switch {
	case portSecurityDisabled && instanceIfacePort.PortSecurityEnabled:
		_, _, err = clientV2.Ports.DisablePortSecurity(ctx, portID)
		if err != nil {
			return diag.FromErr(err)
		}
		return diags
	case !portSecurityDisabled && !instanceIfacePort.PortSecurityEnabled:
		_, _, err = clientV2.Ports.EnablePortSecurity(ctx, portID)
		if err != nil {
			return diag.FromErr(err)
		}
		return diags
	}

	sgsRaw := d.Get(SecurityGroupsField).([]interface{})
	sgs := make([]string, len(sgsRaw), len(sgsRaw))
	for idx, sg := range sgsRaw {
		sgs[idx] = sg.(string)
	}

	filteredSGs, err := utilV2.SecurityGroupListByIDs(ctx, clientV2, sgs)
	if err != nil {
		return diag.FromErr(err)
	}
	sgsNames := make([]string, len(filteredSGs), len(filteredSGs))
	for idx, sg := range filteredSGs {
		sgsNames[idx] = sg.Name
	}

	portSGNames := edgecloudV2.PortsSecurityGroupNames{
		SecurityGroupNames: sgsNames,
		PortID:             portID,
	}

	sgOpts := edgecloudV2.AssignSecurityGroupRequest{PortsSecurityGroupNames: []edgecloudV2.PortsSecurityGroupNames{portSGNames}}

	log.Printf("[DEBUG] attach security group opts: %+v", sgOpts)

	if _, err := clientV2.Instances.SecurityGroupAssign(ctx, instanceID, &sgOpts); err != nil {
		return diag.Errorf("cannot attach security group. Error: %w", err)
	}

	log.Println("[DEBUG] Finish instance_port_security creating")

	return diags
}

func resourceInstancePortSecurityRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start ServerGroup reading")
	var diags diag.Diagnostics

	clientV2, err := InitCloudClient(ctx, d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	portID := d.Get(PortIDField).(string)
	instanceID := d.Get(InstanceIDField).(string)

	instanceIfacePort, err := utilV2.InstanceNetworkInterfaceByID(ctx, clientV2, instanceID, portID)
	if err != nil {
		return diag.FromErr(err)
	}

	sgIDs := make([]string, len(instanceIfacePort.))

	d.Set(PortSecurityDisabledField,!instanceIfacePort.PortSecurityEnabled )
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

func resourceInstancePortSecurityDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
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
