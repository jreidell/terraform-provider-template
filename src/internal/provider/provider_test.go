package provider_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/jreidell/terraform-provider-template/internal/provider"
)

// testAccProtoV6ProviderFactories runs the provider in-process for
// acceptance tests, so no binary has to be built or installed.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"template": providerserver.NewProtocol6WithError(provider.New("test")()),
}

// newFakeAPI starts a stand-in for the UUID API so acceptance tests do not
// depend on the network. Every value it returns is unique.
func newFakeAPI(t *testing.T) *httptest.Server {
	t.Helper()

	var counter atomic.Uint64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count, err := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/count/"))
		if err != nil || count < 1 {
			http.Error(w, "bad count", http.StatusBadRequest)

			return
		}

		uuids := make([]string, count)
		for i := range uuids {
			uuids[i] = fmt.Sprintf(`"00000000-0000-1000-8000-%012d"`, counter.Add(1))
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, "[%s]", strings.Join(uuids, ","))
	}))

	t.Cleanup(server.Close)

	return server
}

func testAccProviderConfig(endpoint string) string {
	return fmt.Sprintf(`
provider "template" {
  endpoint = %[1]q
}
`, endpoint)
}
