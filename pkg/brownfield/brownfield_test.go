package brownfield

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAppgw(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Brownfield Deployment Tests")
}
