package edgecenter

import (
	"context"
	"fmt"
	edgecloud "github.com/Edge-Center/edgecentercloud-go/v2"
	utilV2 "github.com/Edge-Center/edgecentercloud-go/v2/util"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var InstancePortSecNotImplementedErr = fmt.Errorf("instance_port_security are not impelemented yet")

func validatePortSecAttrs(d *schema.ResourceData) diag.Diagnostics {
	diags := diag.Diagnostics{}
	var isPortSecDisabled, isSecGroupExists bool
	if v, ok := d.GetOk(PortSecurityDisabledField); ok {
		isPortSecDisabled = v.(bool)
	}
	_, isSecGroupExists = d.GetOk(SecurityGroupIDsField)
	if isPortSecDisabled && isSecGroupExists {
		curDiag := diag.Diagnostic{
			Severity:      diag.Error,
			Summary:       fmt.Sprintf("if attribute \"%s\" set true, you can't set \"%s\" attribute", PortSecurityDisabledField, SecurityGroupIDsField),
			Detail:        "",
			AttributePath: nil,
		}
		diags = append(diags, curDiag)
	}

	return diags
}

func checkPortSecurityChangesIsApplied(ctx context.Context, d *schema.ResourceData, client *edgecloud.Client) error {
	portID := d.Get(PortIDField).(string)
	instanceID := d.Get(InstanceIDField).(string)
	portSecurityDisabled := d.Get(PortSecurityDisabledField).(bool)
	enforce := d.Get(EnforceField).(bool)
	sgsSet := d.Get(SecurityGroupIDsField).(*schema.Set)

	instancePort, err := utilV2.InstanceNetworkPortByID(ctx, client, instanceID, portID)
	if err != nil {
		return err
	}
	instanceIfacePort, err := utilV2.InstanceNetworkInterfaceByID(ctx, client, instanceID, portID)
	if err != nil {
		return err
	}

	var sgsListFromAPI []interface{}
	for _, sg := range instancePort.SecurityGroups {
		sgsListFromAPI = append(sgsListFromAPI, sg.ID)
	}
	sgsSetFromAPI := schema.NewSet(sgsSet.F, sgsListFromAPI)

	intersectionSet := sgsSetFromAPI.Intersection(sgsSet)
	intersectionDiff := intersectionSet.Difference(sgsSet)

	if intersectionDiff.Len() != 0 {
		return InstancePortSecNotImplementedErr
	}

	if enforce && sgsSet.Len() != intersectionSet.Len() {
		return InstancePortSecNotImplementedErr
	}

	if instanceIfacePort.PortSecurityEnabled == portSecurityDisabled {
		return InstancePortSecNotImplementedErr
	}

	return nil
}
