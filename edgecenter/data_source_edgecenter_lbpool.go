package edgecenter

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	edgecloudV2 "github.com/Edge-Center/edgecentercloud-go/v2"
)

func dataSourceLBPool() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceLBPoolRead,
		Description: "Represent information about load balancer listener pool. A pool is a list of virtual machines to which the listener will redirect incoming traffic.",
		Schema: map[string]*schema.Schema{
			"project_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The uuid of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			"project_name": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the project. Either 'project_id' or 'project_name' must be specified.",
				ExactlyOneOf: []string{"project_id", "project_name"},
			},
			"region_id": {
				Type:         schema.TypeInt,
				Optional:     true,
				Description:  "The uuid of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			"region_name": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "The name of the region. Either 'region_id' or 'region_name' must be specified.",
				ExactlyOneOf: []string{"region_id", "region_name"},
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the load balancer pool.",
			},
			"lb_algorithm": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: fmt.Sprintf("Available values are `%s`, `%s`, `%s`.", edgecloudV2.LoadbalancerAlgorithmRoundRobin, edgecloudV2.LoadbalancerAlgorithmLeastConnections, edgecloudV2.LoadbalancerAlgorithmSourceIP),
			},
			"protocol": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: fmt.Sprintf("Available values are `%s` (currently work, others do not work on ed-8), `%s`, `%s`, `%s`.", edgecloudV2.ListenerProtocolHTTP, edgecloudV2.ListenerProtocolHTTPS, edgecloudV2.ListenerProtocolTCP, edgecloudV2.ListenerProtocolUDP),
			},
			"loadbalancer_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The uuid for the load balancer.",
			},
			"listener_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The uuid for the load balancer listener.",
			},
			"health_monitor": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: `Configuration for health checks to test the health and state of the backend members. It determines how the load balancer identifies whether the backend members are healthy or unhealthy.`,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the health monitor.",
						},
						"type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: fmt.Sprintf("The type of the health monitor. Available values are `%s`, `%s`, `%s`, `%s`, `%s`, `%s`.", edgecloudV2.HealthMonitorTypeHTTP, edgecloudV2.HealthMonitorTypeHTTPS, edgecloudV2.HealthMonitorTypePING, edgecloudV2.HealthMonitorTypeTCP, edgecloudV2.HealthMonitorTypeTLSHello, edgecloudV2.HealthMonitorTypeUDPConnect),
						},
						"delay": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The time between sending probes to members (in seconds).",
						},
						"max_retries": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of successes before the member is switched to the ONLINE state.",
						},
						"timeout": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The maximum time to connect. Must be less than the delay value.",
						},
						"max_retries_down": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of failures before the member is switched to the ERROR state.",
						},
						"http_method": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: fmt.Sprintf("The HTTP method. Available values are `%s`, `%s`, `%s`, `%s`, `%s`, `%s`,`%s`, `%s`, `%s`.", edgecloudV2.HTTPMethodCONNECT, edgecloudV2.HTTPMethodDELETE, edgecloudV2.HTTPMethodGET, edgecloudV2.HTTPMethodHEAD, edgecloudV2.HTTPMethodOPTIONS, edgecloudV2.HTTPMethodPATCH, edgecloudV2.HTTPMethodPOST, edgecloudV2.HTTPMethodPUT, edgecloudV2.HTTPMethodTRACE),
						},
						"url_path": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The URL path. Defaults to `/`.",
						},
						"expected_codes": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The expected HTTP status codes. Multiple codes can be specified as a comma-separated string.",
						},
					},
				},
			},
			"session_persistence": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: `Configuration that enables the load balancer to bind a user's session to a specific backend member. This ensures that all requests from the user during the session are sent to the same member.`,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: fmt.Sprintf("The type of the session persistence. Available values are `%s`,`%s`,`%s`.", edgecloudV2.SessionPersistenceAppCookie, edgecloudV2.SessionPersistenceHTTPCookie, edgecloudV2.SessionPersistenceSourceIP),
						},
						"cookie_name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the cookie. Should be set if app cookie or http cookie is used.",
						},
						"persistence_granularity": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The subnet mask if source_ip is used. For UDP ports only.",
						},
						"persistence_timeout": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The timeout for the session persistence. For UDP ports only.",
						},
					},
				},
			},
		},
	}
}

func dataSourceLBPoolRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	log.Println("[DEBUG] Start LBPool reading")
	var diags diag.Diagnostics
	config := m.(*Config)
	clientV2 := config.CloudClient

	var err error
	clientV2.Region, clientV2.Project, err = GetRegionIDandProjectID(ctx, clientV2, d)
	if err != nil {
		return diag.FromErr(err)
	}

	var opts edgecloudV2.PoolListOptions
	name := d.Get("name").(string)
	lbID := d.Get("loadbalancer_id").(string)
	if lbID != "" {
		opts.LoadbalancerID = lbID
	}
	lID := d.Get("listener_id").(string)
	if lbID != "" {
		opts.ListenerID = lID
	}

	pools, _, err := clientV2.Loadbalancers.PoolList(ctx, &opts)
	if err != nil {
		return diag.FromErr(err)
	}

	var found bool
	var lb edgecloudV2.Pool
	for _, p := range pools {
		if p.Name == name {
			lb = p
			found = true
			break
		}
	}

	if !found {
		return diag.Errorf("lb listener with name %s not found", name)
	}

	d.SetId(lb.ID)
	d.Set("name", lb.Name)
	d.Set("lb_algorithm", lb.LoadbalancerAlgorithm)
	d.Set("protocol", lb.Protocol)

	if len(lb.Loadbalancers) > 0 {
		d.Set("loadbalancer_id", lb.Loadbalancers[0].ID)
	}

	if len(lb.Listeners) > 0 {
		d.Set("listener_id", lb.Listeners[0].ID)
	}

	if lb.HealthMonitor != nil {
		healthMonitor := map[string]interface{}{
			"id":               lb.HealthMonitor.ID,
			"type":             lb.HealthMonitor.Type,
			"delay":            lb.HealthMonitor.Delay,
			"timeout":          lb.HealthMonitor.Timeout,
			"max_retries":      lb.HealthMonitor.MaxRetries,
			"max_retries_down": lb.HealthMonitor.MaxRetriesDown,
			"url_path":         lb.HealthMonitor.URLPath,
			"expected_codes":   lb.HealthMonitor.ExpectedCodes,
		}
		if lb.HealthMonitor.HTTPMethod != nil {
			healthMonitor["http_method"] = lb.HealthMonitor.HTTPMethod
		}

		if err := d.Set("health_monitor", []interface{}{healthMonitor}); err != nil {
			return diag.FromErr(err)
		}
	}

	if lb.SessionPersistence != nil {
		sessionPersistence := map[string]interface{}{
			"type":                    lb.SessionPersistence.Type,
			"cookie_name":             lb.SessionPersistence.CookieName,
			"persistence_granularity": lb.SessionPersistence.PersistenceGranularity,
			"persistence_timeout":     lb.SessionPersistence.PersistenceTimeout,
		}

		if err := d.Set("session_persistence", []interface{}{sessionPersistence}); err != nil {
			return diag.FromErr(err)
		}
	}

	d.Set("project_id", d.Get("project_id").(int))
	d.Set("region_id", d.Get("region_id").(int))

	log.Println("[DEBUG] Finish LBPool reading")

	return diags
}
