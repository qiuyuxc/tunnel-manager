package models

import "testing"

// TestAndroidOriginsEncodeTheSigningCertificate pins the encoding Android's
// Credential Manager reports for an app: the signing certificate's SHA-256,
// base64url encoded and unpadded. Getting it wrong leaves the native app unable
// to register a passkey while the browser keeps working, which is easy to
// mistake for a problem on the device.
func TestAndroidOriginsEncodeTheSigningCertificate(t *testing.T) {
	settings := AppSettings{}
	want := "android:apk-key-hash:_WzJqmT_Wao-ubC2Bj6jeNidjaUukKo8iQLV5M8qSoE"
	got := settings.AndroidOrigins()
	if len(got) != 1 || got[0] != want {
		t.Fatalf("AndroidOrigins() = %#v; want [%q] for the shared debug keystore", got, want)
	}

	// A configured list replaces the default, one origin per fingerprint.
	settings.PasskeyAndroidFingerprints = "01:01:01:01:01:01:01:01:01:01:01:01:01:01:01:01:01:01:01:01:01:01:01:01:01:01:01:01:01:01:01:01"
	got = settings.AndroidOrigins()
	want = "android:apk-key-hash:AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("AndroidOrigins() = %#v; want [%q]", got, want)
	}

	// Unparseable input yields nothing rather than a wrong hash.
	settings.PasskeyAndroidFingerprints = "not-a-fingerprint"
	if got := settings.AndroidOrigins(); len(got) != 0 {
		t.Fatalf("AndroidOrigins() = %#v; want no origins for unparseable input", got)
	}
}
