package coreos

import "testing"

func TestStreamCache(t *testing.T) {
	scache := &StreamCache{
		LocalDir: "testdata/",
	}
	_, err := scache.Get("stable")
	if err != nil {
		t.Errorf("Get got err %q wanted nil", err)
	}
}
