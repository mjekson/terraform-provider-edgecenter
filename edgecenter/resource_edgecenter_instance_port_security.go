package edgecenter

import (
	"context"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"log"
)

const (
	PortSecurityPortIDField   = "port_id"
	PortSecurityDisabledField = "port_security_disabled"
)

func resourceInstancePortSecurity() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceInstancePortSecurityCreate,
		ReadContext:   resourceInstancePortSecurityRead,
		UpdateContext: resourceInstancePortSecurityUpdate,
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
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "ID of the instance to which the port is connected.",
				ValidateFunc: validation.IsUUID,
			},

			PortSecurityDisabledField: {
				Type:        schema.TypeBool,
				Description: "Is the port_security feature disabled.",
				Default:     false,
				Optional:    true,
			},
			PortIDField: {
				Type:         schema.TypeString,
				ForceNew:     true,
				Description:  "ID of the port.",
				Required:     true,
				ValidateFunc: validation.IsUUID,
			},
			EnforceField: {
				Type:        schema.TypeBool,
				Description: "Whether to overwrite all security policies.",
				Optional:    true,
				Default:     false,
			},
			SecurityGroupIDsField: {
				Type:        schema.TypeSet,
				Description: "Set of security groups IDs.",
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			AllSecurityGroupIDsField: {
				Type:        schema.TypeSet,
				Description: "Set of all security groups IDs.",
				Computed:    true,
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
	case !portSecurityDisabled && !instanceIfacePort.PortSecurityEnabled:
		_, _, err = clientV2.Ports.EnablePortSecurity(ctx, portID)
		if err != nil {
			return diag.FromErr(err)
		}
	}
	if portSecurityDisabled {
		d.SetId(portID)

		log.Println("[DEBUG] Finish instance_port_security creating")

		return resourceInstancePortSecurityRead(ctx, d, m)
	}

	enforce := d.Get(EnforceField).(bool)
	sgsRaw, sgsRawOk := d.GetOk(SecurityGroupIDsField)

	if enforce && sgsRawOk {
		var sgsToRemove []interface{}

		instancePort, err := utilV2.InstanceNetworkPortByID(ctx, clientV2, instanceID, portID)
		if err != nil {
			return diag.FromErr(err)
		}
		if len(instancePort.SecurityGroups) != 0 {
			for _, sg := range instancePort.SecurityGroups {
				sgsToRemove = append(sgsToRemove, sg.ID)
			}
			err = removeSecurityGroupsFromInstancePort(ctx, clientV2, instanceID, portID, sgsToRemove)
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}

	if sgsRawOk {
		sgsSet := sgsRaw.(*schema.Set)
		sgsList := sgsSet.List()
		err = AssignSecurityGroupsToInstancePort(ctx, clientV2, instanceID, portID, sgsList)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	d.SetId(portID)

	log.Println("[DEBUG] Finish instance_port_security creating")

	return resourceInstancePortSecurityRead(ctx, d, m)
}

func resourceInstancePortSecurityRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start instance_port_security reading")
	var diags diag.Diagnostics

	clientV2, err := InitCloudClient(ctx, d, m)
	if err != nil {
		return diag.FromErr(err)
	}

	portID := d.Get(PortIDField).(string)
	instanceID := d.Get(InstanceIDField).(string)

	instanceIface, err := utilV2.InstanceNetworkInterfaceByID(ctx, clientV2, instanceID, portID)
	if err != nil {
		return diag.FromErr(err)
	}

	instancePort, err := utilV2.InstanceNetworkPortByID(ctx, clientV2, instanceID, portID)
	if err != nil {
		return diag.FromErr(err)
	}
	d.Set(PortSecurityDisabledField, !instanceIface.PortSecurityEnabled)
	sgsSet := d.Get(SecurityGroupIDsField).(*schema.Set)

	if instanceIface.PortSecurityEnabled {
		sgIDs := make([]interface{}, len(instancePort.SecurityGroups), len(instancePort.SecurityGroups))
		for idx, sg := range instancePort.SecurityGroups {
			sgIDs[idx] = sg.ID
		}
		err = d.Set(AllSecurityGroupIDsField, schema.NewSet(sgsSet.F, sgIDs))
	}

	log.Println("[DEBUG] Finish instance_port_security reading")

	return diags
}

func resourceInstancePortSecurityUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start port_security updating")

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
	portSecurityDisabled := d.Get(PortSecurityDisabledField).(bool)

	if d.HasChange(PortSecurityDisabledField) {
		instanceIfacePort, err := utilV2.InstanceNetworkInterfaceByID(ctx, clientV2, instanceID, portID)
		if err != nil {
			return diag.FromErr(err)
		}

		switch {
		case portSecurityDisabled && instanceIfacePort.PortSecurityEnabled:
			_, _, err = clientV2.Ports.DisablePortSecurity(ctx, portID)
			if err != nil {
				return diag.FromErr(err)
			}
		case !portSecurityDisabled && !instanceIfacePort.PortSecurityEnabled:
			_, _, err = clientV2.Ports.EnablePortSecurity(ctx, portID)
			if err != nil {
				return diag.FromErr(err)
			}
		}
	}
	if portSecurityDisabled {

		log.Println("[DEBUG] Finish instance_port_security updating")

		return resourceInstancePortSecurityRead(ctx, d, m)
	}

	if d.HasChange(SecurityGroupIDsField) || d.HasChange(EnforceField) {
		enforce := d.Get(EnforceField).(bool)
		var sgsToRemoveList []interface{}

		sgsOldRaw, sgsNewRaw := d.GetChange(SecurityGroupIDsField)
		sgsOldSet, sgsNewSet := sgsOldRaw.(*schema.Set), sgsNewRaw.(*schema.Set)

		switch enforce {
		case true:
			allSGIDs := d.Get(AllSecurityGroupIDsField).(*schema.Set)
			sgsToRemoveList = allSGIDs.Difference(sgsNewSet).List()
		default:
			sgsToRemoveList = sgsOldSet.Difference(sgsNewSet).List()
		}

		err = removeSecurityGroupsFromInstancePort(ctx, clientV2, instanceID, portID, sgsToRemoveList)
		if err != nil {
			return diag.FromErr(err)
		}

		sgsToAssignList := sgsNewSet.Difference(sgsOldSet).List()

		err = AssignSecurityGroupsToInstancePort(ctx, clientV2, instanceID, portID, sgsToAssignList)
		if err != nil {
			return diag.FromErr(err)
		}

	}
	err = checkPortSecurityChangesIsApplied(ctx, d, clientV2)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish instance_port_security updating")

	return resourceInstancePortSecurityRead(ctx, d, m)
}

func resourceInstancePortSecurityDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start instance_port_security deleting")
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

	if !instanceIfacePort.PortSecurityEnabled {
		_, _, err = clientV2.Ports.EnablePortSecurity(ctx, portID)
		if err != nil {
			return diag.FromErr(err)
		}
		return diags
	}

	sgsRaw, ok := d.GetOk(SecurityGroupIDsField)
	if !ok {
		return diags
	}
	sgsSet := sgsRaw.(*schema.Set)
	sgsList := sgsSet.List()
	err = removeSecurityGroupsFromInstancePort(ctx, clientV2, instanceID, portID, sgsList)
	if err != nil {
		return diag.FromErr(err)
	}

	log.Println("[DEBUG] Finish instance_port_security deleting")

	return diags
}
