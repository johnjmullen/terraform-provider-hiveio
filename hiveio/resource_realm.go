package hiveio

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hive-io/hive-go-client/rest"
)

func resourceRealm() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRealmCreate,
		ReadContext:   resourceRealmRead,
		UpdateContext: resourceRealmUpdate,
		DeleteContext: resourceRealmDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"fqdn": {
				Description: "fully qualified domain nam",
				Type:        schema.TypeString,
				Required:    true,
			},
			"name": {
				Description: "netbios name",
				Type:        schema.TypeString,
				Required:    true,
			},
			"enabled": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"verified": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"tags": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"username": {
				Type:        schema.TypeString,
				Description: "Service Account username",
				Optional:    true,
			},
			"password": {
				Type:        schema.TypeString,
				Description: "Service Account password",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

func resourceRealmCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*rest.Client)
	realm := &rest.Realm{
		Name: d.Get("name").(string),
		FQDN: d.Get("fqdn").(string),
		ServiceAccount: &rest.RealmServiceAccount{
			Username: d.Get("username").(string),
			Password: d.Get("password").(string),
		},
	}

	_, err := realm.Create(client)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(realm.Name)
	return resourceRealmRead(ctx, d, m)
}

func resourceRealmRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*rest.Client)
	var realm rest.Realm
	var err error
	realm, err = client.GetRealm(d.Id())
	if err != nil && strings.Contains(err.Error(), "\"error\": 404") {
		d.SetId("")
		return diag.Diagnostics{}
	} else if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(realm.Name)
	d.Set("name", realm.Name)
	d.Set("fqdn", realm.FQDN)
	return diag.Diagnostics{}
}

func resourceRealmUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*rest.Client)
	var realm rest.Realm
	realm.Name = d.Get("name").(string)
	realm.FQDN = d.Get("fqdn").(string)
	_, err := realm.Update(client)
	if err != nil {
		return diag.FromErr(err)
	}
	return resourceRealmRead(ctx, d, m)
}

func resourceRealmDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*rest.Client)
	realm, err := client.GetRealm(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	err = realm.Delete(client)
	if err != nil {
		return diag.FromErr(err)
	}
	return diag.Diagnostics{}
}
