package main

import (
	"crypto/tls"
	"testing"
)

func TestR2TransportAvoidsRejectedHandshake(t *testing.T) {
	tr, err := r2Transport()
	if err != nil {
		t.Fatal(err)
	}
	if tr.TLSNextProto == nil {
		t.Fatal("HTTP/2 is still enabled")
	}
	if len(tr.TLSClientConfig.CurvePreferences) == 0 {
		t.Fatal("no curve preferences")
	}
	for _, curve := range tr.TLSClientConfig.CurvePreferences {
		if curve == tls.X25519MLKEM768 {
			t.Fatal("post-quantum key share is still offered")
		}
	}
}

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
