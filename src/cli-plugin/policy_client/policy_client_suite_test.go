package cli_plugin_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCliPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PolicyClient Suite")
}