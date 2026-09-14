package oauth_test

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"squadron/internal/paths"
	"squadron/mcp/oauth"
)

var origDir string

func TestOAuth(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MCP OAuth Suite")
}

var _ = BeforeSuite(func() {
	var err error
	origDir, err = os.Getwd()
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	Expect(os.Chdir(origDir)).To(Succeed())
	paths.ResetHome()
})

// resetRuntimeTokens isolates the process-local OAuth key space between tests.
func resetRuntimeTokens() string {
	dir := GinkgoT().TempDir()
	Expect(os.Chdir(dir)).To(Succeed())
	paths.ResetHome()
	oauth.ClearRuntimeState()
	return dir
}
