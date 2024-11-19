package bytesize

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ByteSize represents a size in bytes with string parsing and formatting
type ByteSize int64

// Common byte sizes for IEC (binary) units
const (
	_            = iota
	KiB ByteSize = 1 << (10 * iota)
	MiB
	GiB
	TiB
	PiB
)

// Common byte sizes for SI (decimal) units
const (
	KB ByteSize = 1000
	MB ByteSize = KB * 1000
	GB ByteSize = MB * 1000
	TB ByteSize = GB * 1000
	PB ByteSize = TB * 1000
)

var (
	ErrInvalidByteSize = errors.New("invalid byte size")
	// Support both IEC and SI units, with optional space and case insensitive
	byteSizeRegex = regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*(?i:([KMGTP]I?B|[KMGTP]|B(?:YTE(?:S)?)?)?)\s*$`)
)

// String returns a human-readable representation of the byte size using IEC units
func (b ByteSize) String() string {
	return b.Format(true)
}

// Format returns a human-readable representation of the byte size
// If useIEC is true, uses binary units (KiB, MiB, etc.)
// If useIEC is false, uses decimal units (KB, MB, etc.)
func (b ByteSize) Format(useIEC bool) string {
	abs := b
	if b < 0 {
		abs = -b
	}

	if useIEC {
		switch {
		case abs >= PiB:
			return fmt.Sprintf("%.2fPiB", float64(b)/float64(PiB))
		case abs >= TiB:
			return fmt.Sprintf("%.2fTiB", float64(b)/float64(TiB))
		case abs >= GiB:
			return fmt.Sprintf("%.2fGiB", float64(b)/float64(GiB))
		case abs >= MiB:
			return fmt.Sprintf("%.2fMiB", float64(b)/float64(MiB))
		case abs >= KiB:
			return fmt.Sprintf("%.2fKiB", float64(b)/float64(KiB))
		default:
			return fmt.Sprintf("%dB", b)
		}
	} else {
		switch {
		case abs >= PB:
			return fmt.Sprintf("%.2fPB", float64(b)/float64(PB))
		case abs >= TB:
			return fmt.Sprintf("%.2fTB", float64(b)/float64(TB))
		case abs >= GB:
			return fmt.Sprintf("%.2fGB", float64(b)/float64(GB))
		case abs >= MB:
			return fmt.Sprintf("%.2fMB", float64(b)/float64(MB))
		case abs >= KB:
			return fmt.Sprintf("%.2fKB", float64(b)/float64(KB))
		default:
			return fmt.Sprintf("%dB", b)
		}
	}
}

// Int64 returns the size as an int64
func (b ByteSize) Int64() int64 {
	return int64(b)
}

// ParseByteSize parses a string representation of bytes into a ByteSize value
func ParseByteSize(s string) (ByteSize, error) {
	if s == "" {
		return 0, ErrInvalidByteSize
	}

	matches := byteSizeRegex.FindStringSubmatch(strings.ToUpper(s))
	if matches == nil {
		return 0, ErrInvalidByteSize
	}

	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, ErrInvalidByteSize
	}

	unit := matches[2]
	if unit == "" || unit == "B" || unit == "BYTE" || unit == "BYTES" {
		return ByteSize(value), nil
	}

	// Check if it's an IEC unit (has 'I' in it)
	isIEC := strings.Contains(unit, "I")
	unitChar := rune(unit[0])

	var multiplier ByteSize
	switch unitChar {
	case 'K':
		multiplier = KiB
		if !isIEC {
			multiplier = KB
		}
	case 'M':
		multiplier = MiB
		if !isIEC {
			multiplier = MB
		}
	case 'G':
		multiplier = GiB
		if !isIEC {
			multiplier = GB
		}
	case 'T':
		multiplier = TiB
		if !isIEC {
			multiplier = TB
		}
	case 'P':
		multiplier = PiB
		if !isIEC {
			multiplier = PB
		}
	default:
		return 0, ErrInvalidByteSize
	}

	return ByteSize(value * float64(multiplier)), nil
}

// MarshalText implements the encoding.TextMarshaler interface
func (b ByteSize) MarshalText() ([]byte, error) {
	return []byte(b.String()), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface
func (b *ByteSize) UnmarshalText(text []byte) error {
	size, err := ParseByteSize(string(text))
	if err != nil {
		return err
	}
	*b = size
	return nil
}
