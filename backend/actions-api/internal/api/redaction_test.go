package api

import (
	"strings"
	"testing"
)

func TestRedactLine_CommandTokens(t *testing.T) {
	p := PrivacySettings{RedactSecrets: true, RedactCmd: true}
	in := `curl -H "Authorization: Bearer abcdefghi123456" --password s3cr3t-token-here`
	out := RedactLine(in, p)
	if out == in {
		t.Fatalf("expected redaction")
	}
	if strings.Contains(out, "abcdef") || strings.Contains(out, "s3cr3t") {
		t.Fatalf("secret leaked: %q", out)
	}
}

func TestRedactLine_IP_MAC_Email_User_Host_Path(t *testing.T) {
	p := PrivacySettings{
		RedactIPs: true, RedactMACs: true, RedactEmails: true, RedactUsers: true, RedactHosts: true, RedactPaths: true,
	}
	in := `User user@host: alice path /home/alice/key src 10.0.0.5 mac aa:bb:cc:dd:ee:ff mail a@b.com host lab.internal`
	out := RedactLine(in, p)
	for _, bad := range []string{"10.0.0.5", "aa:bb:cc:dd:ee:ff", "a@b.com", "/home/alice/key"} {
		if strings.Contains(out, bad) {
			t.Fatalf("leaked %q in %q", bad, out)
		}
	}
}
