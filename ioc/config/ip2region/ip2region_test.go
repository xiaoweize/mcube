package ip2region_test

import (
	"testing"

	"github.com/infraboard/mcube/v2/ioc"
	"github.com/infraboard/mcube/v2/ioc/config/ip2region"
)

func TestSearch(t *testing.T) {
	resp, err := ip2region.Get().LookupIP("117.136.38.42")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(resp)
}

func init() {
	ioc.DevelopmentSetup()
}
