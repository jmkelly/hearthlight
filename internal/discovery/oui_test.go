package discovery

import (
	"strconv"
	"strings"
	"testing"
)

// Duplicate OUI keys are a compile error in a map literal, so uniqueness is
// enforced by the compiler. This test guards the key shape and vendor values.
func TestOUITableShape(t *testing.T) {
	for key, vendor := range ouiVendors {
		parts := strings.Split(key, ":")
		if len(key) != 8 || len(parts) != 3 {
			t.Errorf("OUI key %q is not xx:xx:xx", key)
			continue
		}
		for _, part := range parts {
			if len(part) != 2 || part != strings.ToLower(part) {
				t.Errorf("OUI key %q is not lower-case xx:xx:xx", key)
				break
			}
			if _, err := strconv.ParseUint(part, 16, 8); err != nil {
				t.Errorf("OUI key %q contains a non-hex octet", key)
				break
			}
		}
		if vendor == "" {
			t.Errorf("OUI key %q has an empty vendor", key)
		}
	}
}

func TestVendorForMACAddedPrefixes(t *testing.T) {
	tests := []struct {
		mac  string
		want string
	}{
		// MikroTik
		{"00:0c:42:aa:bb:cc", "MikroTik"},
		{"08:55:31:aa:bb:cc", "MikroTik"},
		{"2c:c8:1b:aa:bb:cc", "MikroTik"},
		{"48:8f:5a:aa:bb:cc", "MikroTik"},
		{"4c:5e:0c:aa:bb:cc", "MikroTik"},
		{"6c:3b:6b:aa:bb:cc", "MikroTik"},
		// Roku
		{"00:0d:4b:aa:bb:cc", "Roku"},
		{"b0:a7:37:aa:bb:cc", "Roku"},
		{"cc:6d:a0:aa:bb:cc", "Roku"},
		{"d8:31:34:aa:bb:cc", "Roku"},
		{"dc:3a:5e:aa:bb:cc", "Roku"},
		// Microsoft
		{"00:0d:3a:aa:bb:cc", "Microsoft"},
		{"00:12:5a:aa:bb:cc", "Microsoft"},
		{"00:15:5d:aa:bb:cc", "Microsoft"},
		{"00:50:f2:aa:bb:cc", "Microsoft"},
		{"28:18:78:aa:bb:cc", "Microsoft"},
		{"7c:1e:52:aa:bb:cc", "Microsoft"},
		// Huawei
		{"00:18:82:aa:bb:cc", "Huawei"},
		{"00:1e:10:aa:bb:cc", "Huawei"},
		{"00:25:9e:aa:bb:cc", "Huawei"},
		{"04:bd:70:aa:bb:cc", "Huawei"},
		{"08:19:a6:aa:bb:cc", "Huawei"},
		{"18:c5:8a:aa:bb:cc", "Huawei"},
		// Tuya
		{"10:d5:61:aa:bb:cc", "Tuya"},
		{"18:69:d8:aa:bb:cc", "Tuya"},
		{"18:de:50:aa:bb:cc", "Tuya"},
		{"44:67:55:aa:bb:cc", "Tuya"},
		{"50:02:91:aa:bb:cc", "Tuya"},
		{"68:57:2d:aa:bb:cc", "Tuya"},
		// Philips
		{"00:1f:e2:aa:bb:cc", "Philips"},
		// Apple
		{"00:03:93:aa:bb:cc", "Apple"},
		{"00:11:24:aa:bb:cc", "Apple"},
		{"00:16:cb:aa:bb:cc", "Apple"},
		{"00:1e:c2:aa:bb:cc", "Apple"},
		{"04:0c:ce:aa:bb:cc", "Apple"},
		{"3c:07:54:aa:bb:cc", "Apple"},
		{"5c:95:ae:aa:bb:cc", "Apple"},
		{"8c:85:90:aa:bb:cc", "Apple"},
		// Google
		{"00:1a:11:aa:bb:cc", "Google"},
		{"30:fd:38:aa:bb:cc", "Google"},
		{"48:d6:d5:aa:bb:cc", "Google"},
		{"a4:77:33:aa:bb:cc", "Google"},
		// Amazon
		{"74:c2:46:aa:bb:cc", "Amazon"},
		{"78:e1:03:aa:bb:cc", "Amazon"},
		{"a0:02:dc:aa:bb:cc", "Amazon"},
		{"ac:63:be:aa:bb:cc", "Amazon"},
		// Samsung
		{"00:12:fb:aa:bb:cc", "Samsung"},
		{"5c:0a:5b:aa:bb:cc", "Samsung"},
		{"78:1f:db:aa:bb:cc", "Samsung"},
		{"c0:bd:d1:aa:bb:cc", "Samsung"},
		// Intel
		{"00:1b:21:aa:bb:cc", "Intel"},
		{"00:21:5c:aa:bb:cc", "Intel"},
		{"3c:97:0e:aa:bb:cc", "Intel"},
		{"a0:36:9f:aa:bb:cc", "Intel"},
		// Espressif
		{"24:62:ab:aa:bb:cc", "Espressif"},
		{"2c:3a:e8:aa:bb:cc", "Espressif"},
		{"3c:61:05:aa:bb:cc", "Espressif"},
		{"5c:cf:7f:aa:bb:cc", "Espressif"},
		{"60:01:94:aa:bb:cc", "Espressif"},
		{"a0:20:a6:aa:bb:cc", "Espressif"},
	}
	for _, tt := range tests {
		t.Run(tt.mac, func(t *testing.T) {
			if got := vendorForMAC(tt.mac); got != tt.want {
				t.Fatalf("vendorForMAC(%q) = %q, want %q", tt.mac, got, tt.want)
			}
		})
	}
}

func TestVendorForMACExistingPrefixes(t *testing.T) {
	tests := []struct {
		mac  string
		want string
	}{
		{"00:1b:63:aa:bb:cc", "Apple"},
		{"b8:27:eb:aa:bb:cc", "Raspberry Pi"},
		{"24:0a:c4:aa:bb:cc", "Espressif"},
		{"3c:58:c2:aa:bb:cc", "Intel"},
		{"8c:77:12:aa:bb:cc", "Samsung"},
		{"70:3a:cb:aa:bb:cc", "Google"},
		{"00:11:32:aa:bb:cc", "Synology"},
		{"00:08:9b:aa:bb:cc", "QNAP"},
		{"00:17:88:aa:bb:cc", "Philips Hue"},
		{"00:1f:29:aa:bb:cc", "HP"},
		{"00:14:22:aa:bb:cc", "Dell"},
		{"dc:cd:2f:aa:bb:cc", "Epson"},
		{"00:25:90:aa:bb:cc", "Supermicro"},
		{"20:4e:7f:aa:bb:cc", "Netgear"},
		{"04:d4:c4:aa:bb:cc", "ASUS"},
	}
	for _, tt := range tests {
		t.Run(tt.mac, func(t *testing.T) {
			if got := vendorForMAC(tt.mac); got != tt.want {
				t.Fatalf("vendorForMAC(%q) = %q, want %q", tt.mac, got, tt.want)
			}
		})
	}
}

func TestVendorForMACNormalizesInput(t *testing.T) {
	tests := []struct {
		mac  string
		want string
	}{
		{"  00:0C:42:AA:BB:CC  ", "MikroTik"},
		{"00:0c", ""},
		{"", ""},
	}
	for _, tt := range tests {
		if got := vendorForMAC(tt.mac); got != tt.want {
			t.Errorf("vendorForMAC(%q) = %q, want %q", tt.mac, got, tt.want)
		}
	}
}
