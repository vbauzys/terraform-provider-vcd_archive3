// +build vapp ALL functional

package vcd

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"github.com/vmware/go-vcloud-director/v2/govcd"
)

var vappName = "TestAccVcdVAppVapp"

func TestAccVcdVApp_Basic(t *testing.T) {
	var vapp govcd.VApp

	var params = StringMap{
		"Org":          testConfig.VCD.Org,
		"Vdc":          testConfig.VCD.Vdc,
		"EdgeGateway":  testConfig.Networking.EdgeGateway,
		"NetworkName":  "TestAccVcdVAppNet",
		"NetworkName2": "TestAccVcdVAppNet2",
		"NetworkName3": "TestAccVcdVAppNet3",
		"Catalog":      testSuiteCatalogName,
		"CatalogItem":  testSuiteCatalogOVAItem,
		"VappName":     vappName,
		"FuncName":     "TestAccCheckVcdVApp_PowerOff",
		"Tags":         "vapp",
	}
	configText := templateFill(testAccCheckVcdVApp_basic, params)

	params["FuncName"] = "TestAccCheckVcdVApp_powerOff"

	configTextUpdate := templateFill(testAccCheckVcdVApp_update, params)
	if vcdShortTest {
		t.Skip(acceptanceTestsSkipped)
		return
	}
	debugPrintf("#[DEBUG] CONFIGURATION basic: %s\n", configText)
	debugPrintf("#[DEBUG] CONFIGURATION udpate: %s\n", configTextUpdate)
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckVcdVAppDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: configText,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVcdVAppExists("vcd_vapp."+vappName, &vapp),
					resource.TestCheckResourceAttr("vcd_vapp."+vappName, "name", vappName),
					resource.TestCheckResourceAttr("vcd_vapp."+vappName, "metadata.vapp_metadata", "vApp Metadata."),
					resource.TestMatchResourceAttr("vcd_vapp."+vappName, "href",
						regexp.MustCompile(`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`)),
					resource.TestCheckResourceAttr("vcd_vapp."+vappName, "name", vappName),
					resource.TestCheckResourceAttr("vcd_vapp."+vappName, "metadata.vapp_metadata", "vApp Metadata."),
					resource.TestCheckResourceAttr("vcd_vapp."+vappName, `guest_properties.guest.hostname`, "test-host"),
					resource.TestCheckResourceAttr("vcd_vapp."+vappName, `guest_properties.guest.another.subkey`, "another-value"),
				),
			},
			resource.TestStep{
				Config: configTextUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVcdVAppExists("vcd_vapp."+vappName, &vapp),
					resource.TestCheckResourceAttr("vcd_vapp."+vappName, "name", vappName),
					resource.TestCheckResourceAttr("vcd_vapp."+vappName, "metadata.vapp_metadata", "vApp Metadata updated"),
					resource.TestCheckResourceAttr("vcd_vapp."+vappName, `guest_properties.guest.another.subkey`, "new-value"),
					resource.TestCheckResourceAttr("vcd_vapp."+vappName, `guest_properties.guest.third.subkey`, "third-value"),
				),
			},
			resource.TestStep{
				ResourceName:      "vcd_vapp." + vappName + "-import",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: importStateIdOrgVdcObject(testConfig, vappName),
				// These fields can't be retrieved from user data
				ImportStateVerifyIgnore: []string{},
			},
		},
	})
}

func testAccCheckVcdVAppExists(n string, vapp *govcd.VApp) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no vApp ID is set")
		}

		conn := testAccProvider.Meta().(*VCDClient)

		_, vdc, err := conn.GetOrgAndVdc(testConfig.VCD.Org, testConfig.VCD.Vdc)
		if err != nil {
			return fmt.Errorf(errorRetrievingVdcFromOrg, testConfig.VCD.Vdc, testConfig.VCD.Org, err)
		}

		newVapp, err := vdc.GetVAppByNameOrId(rs.Primary.ID, false)
		if err != nil {
			return err
		}

		*vapp = *newVapp

		return nil
	}
}

func testAccCheckVcdVAppDestroy(s *terraform.State) error {
	conn := testAccProvider.Meta().(*VCDClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "vcd_vapp" {
			continue
		}

		_, vdc, err := conn.GetOrgAndVdc(testConfig.VCD.Org, testConfig.VCD.Vdc)
		if err != nil {
			return fmt.Errorf(errorRetrievingVdcFromOrg, testConfig.VCD.Vdc, testConfig.VCD.Org, err)
		}

		_, err = vdc.GetVAppByNameOrId(rs.Primary.ID, false)

		if err == nil {
			return fmt.Errorf("VPCs still exist")
		}

		return nil
	}

	return nil
}

func init() {
	testingTags["vapp"] = "resource_vcd_vapp_test.go"
}

const testAccCheckVcdVApp_basic = `

resource "vcd_vapp" "{{.VappName}}" {
  org           = "{{.Org}}"
  vdc           = "{{.Vdc}}"
  name          = "{{.VappName}}"

  metadata = {
    vapp_metadata = "vApp Metadata."
  }

  guest_properties = {
	"guest.hostname"       = "test-host"
	"guest.another.subkey" = "another-value"
  }
}
`

const testAccCheckVcdVApp_update = `
# skip-binary-test: only for updates
resource "vcd_vapp" "{{.VappName}}" {
  org           = "{{.Org}}"
  vdc           = "{{.Vdc}}"
  name          = "{{.VappName}}"

  metadata = {
    vapp_metadata = "vApp Metadata updated"
  }

  guest_properties = {
	"guest.another.subkey" = "new-value"
	"guest.third.subkey"   = "third-value"
  }
}
`
