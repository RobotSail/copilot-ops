package cmd_test

import (
	"log"
	"net/http/httptest"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/redhat-et/copilot-ops/pkg/ai"
	"github.com/redhat-et/copilot-ops/pkg/ai/gpt3"
	"github.com/redhat-et/copilot-ops/pkg/cmd"
	"github.com/spf13/cobra"
)

var _ = Describe("Edit", func() {
	var c *cobra.Command

	BeforeEach(func() {
		// create command
		c = cmd.NewEditCmd()
		Expect(c).NotTo(BeNil())
	})
	AfterEach(func() {
		os.Clearenv()
	})

	When("OpenAI server exists", func() {
		var ts *httptest.Server
		BeforeEach(func() {
			ts = OpenAITestServer()
		})

		JustBeforeEach(func() {
			ts.Start()
			gpt3.DefaultConfig.BaseURL = ts.URL + gpt3.OpenAIEndpointV1
			os.Setenv("COPILOT_OPS_GPT_3_URL", ts.URL+gpt3.OpenAIEndpointV1)
			log.Printf("current base URL: %q\n", gpt3.DefaultConfig.BaseURL)
		})

		AfterEach(func() {
			defer ts.Close()
		})

		It("works", func() {
			log.Printf("requesting the following url: %q\n", ts.URL)
			err := c.Flags().Set(cmd.FlagAIBackendFull, string(ai.GPT3))
			Expect(err).NotTo(HaveOccurred())
			err = cmd.RunEdit(c, []string{})
			Expect(err).To(BeNil())
		})

	})
})
