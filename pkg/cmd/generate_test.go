package cmd_test

import (
	"log"
	"net/http/httptest"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	"github.com/redhat-et/copilot-ops/pkg/ai"
	"github.com/redhat-et/copilot-ops/pkg/ai/gpt3"
	"github.com/redhat-et/copilot-ops/pkg/cmd"
)

var _ = Describe("Generate command", func() {
	var c *cobra.Command
	var ts *httptest.Server

	BeforeEach(func() {
		c = cmd.NewGenerateCmd()
	})

	AfterEach(func() {
		os.Clearenv()
	})
	When("server is created", func() {
		BeforeEach(func() {
			ts = OpenAITestServer()

			Expect(c).NotTo(BeNil())
			err := c.Flags().Set(cmd.FlagNTokensFull, "1")
			Expect(err).To(BeNil())
		})

		JustBeforeEach(func() {
			ts.Start()
			log.Printf("using server URL: %q\n", ts.URL)
			customURL := ts.URL + gpt3.OpenAIEndpointV1
			Expect(customURL).NotTo(BeEmpty())
			gpt3.DefaultConfig.BaseURL = customURL
			err := c.Flags().Set(cmd.FlagAIBackendFull, string(ai.GPT3))
			Expect(err).To(BeNil())
		})
		AfterEach(func() {
			ts.Close()
		})

		It("executes properly", func() {
			err := cmd.RunGenerate(c, []string{})
			// use the minimum amount of tokens from OpenAI
			Expect(err).To(BeNil())
		})
		// TODO: add more tests for expected success
	})

	When("OpenAI server is down", func() {
		BeforeEach(func() {
			// set a port that isn't taken
			os.Setenv("COPILOT_OPS_GPT_3_URL", "http://localhost:23423")
		})
		It("fails", func() {
			err := cmd.RunGenerate(c, []string{})
			Expect(err).To(HaveOccurred())
		})
		// TODO: add more cases that should fail
	})
})
