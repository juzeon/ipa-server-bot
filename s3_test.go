package main

import "testing"

func TestApplicationObjectKeys(t *testing.T) {
	id := "11111111-1111-1111-1111-111111111111"
	keys, err := applicationObjectKeys(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 3 || keys[0] != id+"/0.ipa" || keys[1] != id+"/manifest.plist" || keys[2] != id+"/install.html" {
		t.Fatalf("keys = %#v", keys)
	}
	for _, id := range []string{"../x", id + "/evil", "not-a-uuid", ""} {
		if _, err := applicationObjectKeys(id); err == nil {
			t.Fatalf("accepted %q", id)
		}
	}
}
