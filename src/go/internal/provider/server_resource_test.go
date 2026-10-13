package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

const serverAddress = "template_server.test"

func TestAccServerResource(t *testing.T) {
	api := newFakeAPI(t)

	// Each tracker asserts how a value moves across the steps it is added to.
	idUnchanged := statecheck.CompareValue(compare.ValuesSame())
	uuidsOnCountChange := statecheck.CompareValue(compare.ValuesDiffer())
	uuidsOnDelayChange := statecheck.CompareValue(compare.ValuesSame())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create with the default delays.
			{
				Config: testAccServerConfig(api.URL, 2, ""),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(serverAddress, tfjsonpath.New("id"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(serverAddress, tfjsonpath.New("uuid_count"), knownvalue.Int64Exact(2)),
					statecheck.ExpectKnownValue(serverAddress, tfjsonpath.New("uuids"), knownvalue.ListSizeExact(2)),
					statecheck.ExpectKnownValue(serverAddress, tfjsonpath.New("creation_delay"), knownvalue.Int64Exact(0)),
					statecheck.ExpectKnownValue(serverAddress, tfjsonpath.New("update_delay"), knownvalue.Int64Exact(0)),
					statecheck.ExpectKnownValue(serverAddress, tfjsonpath.New("deletion_delay"), knownvalue.Int64Exact(0)),
					idUnchanged.AddStateValue(serverAddress, tfjsonpath.New("id")),
					uuidsOnCountChange.AddStateValue(serverAddress, tfjsonpath.New("uuids")),
				},
			},
			// Changing uuid_count updates in place and regenerates the UUIDs.
			{
				Config: testAccServerConfig(api.URL, 3, ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(serverAddress, plancheck.ResourceActionUpdate),
						plancheck.ExpectUnknownValue(serverAddress, tfjsonpath.New("uuids")),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(serverAddress, tfjsonpath.New("uuids"), knownvalue.ListSizeExact(3)),
					idUnchanged.AddStateValue(serverAddress, tfjsonpath.New("id")),
					uuidsOnCountChange.AddStateValue(serverAddress, tfjsonpath.New("uuids")),
					uuidsOnDelayChange.AddStateValue(serverAddress, tfjsonpath.New("uuids")),
				},
			},
			// Changing only a delay updates in place and keeps the UUIDs.
			{
				Config: testAccServerConfig(api.URL, 3, "update_delay = 1"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(serverAddress, plancheck.ResourceActionUpdate),
						plancheck.ExpectKnownValue(serverAddress, tfjsonpath.New("uuids"), knownvalue.ListSizeExact(3)),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(serverAddress, tfjsonpath.New("update_delay"), knownvalue.Int64Exact(1)),
					idUnchanged.AddStateValue(serverAddress, tfjsonpath.New("id")),
					uuidsOnDelayChange.AddStateValue(serverAddress, tfjsonpath.New("uuids")),
				},
			},
			// Delete is exercised automatically when the test case ends.
		},
	})
}

func TestAccServerResource_validation(t *testing.T) {
	api := newFakeAPI(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccServerConfig(api.URL, 101, ""),
				ExpectError: regexp.MustCompile(`uuid_count value must be between 1 and 100`),
			},
			{
				Config:      testAccServerConfig(api.URL, 1, "creation_delay = -1"),
				ExpectError: regexp.MustCompile(`creation_delay value must be at least 0`),
			},
		},
	})
}

func testAccServerConfig(endpoint string, uuidCount int, extra string) string {
	return testAccProviderConfig(endpoint) + fmt.Sprintf(`
resource "template_server" "test" {
  uuid_count = %[1]d
  %[2]s
}
`, uuidCount, extra)
}
