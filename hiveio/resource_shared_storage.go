package hiveio

import (
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hive-io/hive-go-client/rest"
)

var storage_pool_name = "HF_Shared"

func resourceSharedStorage() *schema.Resource {
	return &schema.Resource{
		Create: resourceSharedStorageCreate,
		Read:   resourceSharedStorageRead,
		Delete: resourceSharedStorageDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"minimum_set_size": {
				Type:        schema.TypeInt,
				Description: "minimum number of hosts required to increase shared storage",
				Default:     3,
				Optional:    true,
				ForceNew:    true,
			},
			"utilization": {
				Type:     schema.TypeInt,
				Default:  75,
				Optional: true,
				ForceNew: true,
			},
			"name": {
				Type:        schema.TypeString,
				Description: "storage pool name",
				Computed:    true,
			},
			"type": {
				Type:        schema.TypeString,
				Description: "storage pool type",
				Computed:    true,
			},
			"hosts": {
				Type:        schema.TypeList,
				Description: "helper field to add a dependency on hosts which are added to the cluster at the same time",
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				ForceNew: true,
			},
		},
		Timeouts: &schema.ResourceTimeout{
			Delete: schema.DefaultTimeout(3 * time.Minute),
		},
	}
}

func resourceSharedStorageCreate(d *schema.ResourceData, m interface{}) error {
	client := m.(*rest.Client)
	setSize := d.Get("minimum_set_size").(int)
	utilization := d.Get("utilization").(int)
	clusterID, err := client.ClusterID()
	if err != nil {
		return err
	}
	cluster, err := client.GetCluster(clusterID)
	if err != nil {
		return err
	}
	err = resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		task, err := cluster.EnableSharedStorage(client, utilization, setSize)
		if err != nil && strings.Contains(err.Error(), "Not enough hosts") {
			//waitForMinimumHosts(client, clusterID, setSize, 30*time.Second)
			time.Sleep(15 * time.Second)
			return resource.RetryableError(fmt.Errorf("not enough hosts"))
		}
		task, err = task.WaitForTask(client, false)
		if err != nil {
			return resource.NonRetryableError(err)
		}
		if task.State == "failed" {
			return resource.NonRetryableError(fmt.Errorf("failed to Enable Shared storage: %s", task.Message))
		}
		return nil
	})
	if err != nil {
		return err
	}
	cluster, err = client.GetCluster(clusterID)
	if err != nil {
		return err
	}
	storage, err := client.GetStoragePool(cluster.SharedStorage.ID)
	if err != nil {
		return fmt.Errorf("storage pool not found in database")
	}
	d.SetId(storage.ID)
	d.Set("name", storage.Name)
	d.Set("type", storage.Type)
	return resourceSharedStorageRead(d, m)
}

func resourceSharedStorageRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*rest.Client)
	clusterID, err := client.ClusterID()
	if err != nil {
		return err
	}
	cluster, err := client.GetCluster(clusterID)
	if err != nil {
		return err
	}
	if cluster.SharedStorage == nil || cluster.SharedStorage.ID == "" {
		d.SetId("")
		return nil
	}
	storage, err := client.GetStoragePool(cluster.SharedStorage.ID)
	if err != nil {
		return err
	}
	d.SetId(storage.ID)
	d.Set("name", storage.Name)
	d.Set("type", storage.Type)
	return nil
}

func resourceSharedStorageDelete(d *schema.ResourceData, m interface{}) error {
	client := m.(*rest.Client)
	return resource.Retry(d.Timeout(schema.TimeoutDelete), func() *resource.RetryError {
		clusterID, err := client.ClusterID()
		if err != nil {
			return resource.NonRetryableError(err)
		}
		cluster, err := client.GetCluster(clusterID)
		if err != nil {
			return resource.NonRetryableError(err)
		}
		task, err := cluster.DisableSharedStorage(client)
		if err != nil {
			return resource.RetryableError(err)
		}
		task, err = task.WaitForTask(client, false)
		if err != nil {
			return resource.RetryableError(err)
		}
		if task.State == "failed" {
			return resource.NonRetryableError(fmt.Errorf("failed to Disable Shared storage: %s", task.Message))
		}
		return nil
	})
}
