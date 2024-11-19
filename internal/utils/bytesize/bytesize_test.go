package bytesize

import (
	"testing"
)

func TestParseByteSizeValid(t *testing.T) {
	tests := []struct {
		input    string
		expected ByteSize
	}{
		// Basic byte values
		{"0", 0},
		{"1024", 1024},
		{"1024B", 1024},
		{"1024 B", 1024},
		{"1024 BYTES", 1024},
		{"1024 BYTE", 1024},

		// IEC units (binary)
		{"1KiB", KiB},
		{"1 KiB", KiB},
		{"1.5KiB", ByteSize(float64(KiB) * 1.5)},
		{"1MiB", MiB},
		{"1.5MiB", ByteSize(float64(MiB) * 1.5)},
		{"1GiB", GiB},
		{"1TiB", TiB},
		{"1PiB", PiB},

		// SI units (decimal)
		{"1KB", KB},
		{"1 KB", KB},
		{"1.5KB", ByteSize(float64(KB) * 1.5)},
		{"1MB", MB},
		{"1.5MB", ByteSize(float64(MB) * 1.5)},
		{"1GB", GB},
		{"1TB", TB},
		{"1PB", PB},

		// Short forms (default to SI units)
		{"1K", KB}, // Defaults to SI
		{"1M", MB},
		{"1G", GB},
		{"1T", TB},
		{"1P", PB},

		// Case insensitivity
		{"1kb", KB},
		{"1kib", KiB},
		{"1mB", MB},
		{"1mIb", MiB},
		{"1Kb", KB},
		{"1KiB", KiB},

		// With spaces
		{"1 KB", KB},
		{"1 KiB", KiB},
		{"1 MB", MB},
		{"1 MiB", MiB},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result, err := ParseByteSize(test.input)
			if err != nil {
				t.Errorf("ParseByteSize(%q) returned error: %v", test.input, err)
			}
			if result != test.expected {
				t.Errorf("ParseByteSize(%q) = %v, want %v", test.input, result, test.expected)
			}
		})
	}
}

func TestParseByteSizeInvalid(t *testing.T) {
	tests := []string{
		"",
		"abc",
		"1XB",
		"1.5.5MB",
		"-KB",
		"KB",
		"1KB1",
		"1.KB",
		".5KB",
	}

	for _, test := range tests {
		t.Run(test, func(t *testing.T) {
			_, err := ParseByteSize(test)
			if err == nil {
				t.Errorf("ParseByteSize(%q) should have returned an error", test)
			}
		})
	}
}

func TestByteSizeString(t *testing.T) {
	tests := []struct {
		input    ByteSize
		expected string
	}{
		{0, "0B"},
		{512, "512B"},
		{KiB, "1.00KiB"},
		{ByteSize(float64(KiB) * 1.5), "1.50KiB"},
		{MiB, "1.00MiB"},
		{ByteSize(float64(MiB) * 2.25), "2.25MiB"},
		{GiB, "1.00GiB"},
		{TiB, "1.00TiB"},
		{PiB, "1.00PiB"},
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			result := test.input.String()
			if result != test.expected {
				t.Errorf("ByteSize(%d).String() = %q, want %q", test.input, result, test.expected)
			}
		})
	}
}

func TestByteSizeFormat(t *testing.T) {
	tests := []struct {
		input    ByteSize
		useIEC   bool
		expected string
	}{
		// IEC (binary) format
		{KiB, true, "1.00KiB"},
		{MiB, true, "1.00MiB"},
		{GiB, true, "1.00GiB"},
		{ByteSize(float64(KiB) * 1.5), true, "1.50KiB"},

		// SI (decimal) format
		{KB, false, "1.00KB"},
		{MB, false, "1.00MB"},
		{GB, false, "1.00GB"},
		{ByteSize(float64(KB) * 1.5), false, "1.50KB"},

		// Edge cases
		{0, true, "0B"},
		{0, false, "0B"},
		{512, true, "512B"},
		{512, false, "512B"},
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			result := test.input.Format(test.useIEC)
			if result != test.expected {
				t.Errorf("ByteSize(%d).Format(%v) = %q, want %q",
					test.input, test.useIEC, result, test.expected)
			}
		})
	}
}

func TestTextMarshaling(t *testing.T) {
	tests := []struct {
		size     ByteSize
		expected string
	}{
		{KiB, "1.00KiB"},
		{MiB, "1.00MiB"},
		{ByteSize(float64(GiB) * 1.5), "1.50GiB"},
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			// Test marshaling
			bytes, err := test.size.MarshalText()
			if err != nil {
				t.Errorf("MarshalText() returned error: %v", err)
			}
			if string(bytes) != test.expected {
				t.Errorf("MarshalText() = %q, want %q", string(bytes), test.expected)
			}

			// Test unmarshaling
			var size ByteSize
			err = size.UnmarshalText([]byte(test.expected))
			if err != nil {
				t.Errorf("UnmarshalText(%q) returned error: %v", test.expected, err)
			}
			if size != test.size {
				t.Errorf("UnmarshalText(%q) = %v, want %v", test.expected, size, test.size)
			}
		})
	}
}
