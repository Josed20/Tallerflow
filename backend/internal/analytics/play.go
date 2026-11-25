package analytics

import (
	"bytes"
	"encoding/binary"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"
)

const PlayPackage = "pe.tallerflow.app"
const MaxCSVBytes = 4 << 20

type PlayRow struct {
	HasDeviceInstalls bool
	HasUserInstalls   bool
	HasUninstalls     bool
	Date              time.Time
	Version           string
	Dimension         string
	DeviceInstalls    *int64
	UserInstalls      *int64
	Uninstalls        *int64
}

// ParsePlayCSV accepts the official installs export, including UTF-16 BOMs.
// Only totals or App Version Code reports are accepted: importing countries
// alongside versions would count the same installations more than once.
func ParsePlayCSV(data []byte, now time.Time) ([]PlayRow, error) {
	if len(data) > MaxCSVBytes {
		return nil, errors.New("CSV exceeds 4 MiB")
	}
	if bytes.HasPrefix(data, []byte{0xff, 0xfe}) || bytes.HasPrefix(data, []byte{0xfe, 0xff}) {
		if len(data)%2 != 0 {
			return nil, errors.New("invalid UTF-16 CSV")
		}
		var order binary.ByteOrder = binary.LittleEndian
		if data[0] == 0xfe {
			order = binary.BigEndian
		}
		units := make([]uint16, (len(data)-2)/2)
		for i := range units {
			units[i] = order.Uint16(data[2+i*2:])
		}
		data = []byte(string(utf16.Decode(units)))
	}
	if !utf8.Valid(data) {
		return nil, errors.New("CSV must be UTF-8 or UTF-16 with BOM")
	}
	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(data), "\ufeff")))
	headers, err := reader.Read()
	if err != nil {
		return nil, errors.New("CSV header required")
	}
	columns := map[string]int{}
	metrics := map[string]bool{"Daily Device Installs": true, "Daily User Installs": true, "Daily Device Uninstalls": true}
	known := map[string]bool{"Date": true, "Package Name": true, "App Version Code": true, "Current Device Installs": true, "Installs on active devices": true, "Daily Device Upgrades": true, "Current User Installs": true, "Total User Installs": true, "Daily User Uninstalls": true}
	for i, h := range headers {
		h = strings.TrimSpace(h)
		if _, ok := columns[h]; ok {
			return nil, fmt.Errorf("duplicate column %s", h)
		}
		if !known[h] && !metrics[h] {
			return nil, fmt.Errorf("unsupported column/dimension %s; use totals or App Version Code", h)
		}
		columns[h] = i
	}
	for _, name := range []string{"Date", "Package Name"} {
		if _, ok := columns[name]; !ok {
			return nil, fmt.Errorf("missing %s", name)
		}
	}
	hasMetric := false
	for name := range metrics {
		if _, ok := columns[name]; ok {
			hasMetric = true
		}
	}
	if !hasMetric {
		return nil, errors.New("no supported install metrics")
	}
	value := func(record []string, name string) string {
		if i, ok := columns[name]; ok {
			return strings.TrimSpace(record[i])
		}
		return ""
	}
	rows := []PlayRow{}
	seen := map[string]bool{}
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("invalid CSV row: %w", err)
		}
		if len(rows) >= 10000 {
			return nil, errors.New("CSV exceeds 10000 rows")
		}
		day, err := time.Parse("2006-01-02", value(record, "Date"))
		if err != nil || day.After(now.UTC().Truncate(24*time.Hour)) || day.Before(now.AddDate(0, -25, 0).UTC().Truncate(24*time.Hour)) {
			return nil, errors.New("date outside retained period")
		}
		if value(record, "Package Name") != PlayPackage {
			return nil, errors.New("unexpected Android package")
		}
		row := PlayRow{Date: day, Version: "all", Dimension: "all"}
		_, row.HasDeviceInstalls = columns["Daily Device Installs"]
		_, row.HasUserInstalls = columns["Daily User Installs"]
		_, row.HasUninstalls = columns["Daily Device Uninstalls"]
		if _, ok := columns["App Version Code"]; ok {
			row.Version = value(record, "App Version Code")
			row.Dimension = "app_version"
			code, err := strconv.ParseUint(row.Version, 10, 32)
			if err != nil || code == 0 {
				return nil, errors.New("invalid App Version Code")
			}
			row.Version = strconv.FormatUint(code, 10)
		}
		key := day.Format("2006-01-02") + "/" + row.Version
		if seen[key] {
			return nil, errors.New("duplicate date/version")
		}
		seen[key] = true
		for name, dest := range map[string]**int64{"Daily Device Installs": &row.DeviceInstalls, "Daily User Installs": &row.UserInstalls, "Daily Device Uninstalls": &row.Uninstalls} {
			raw := value(record, name)
			if raw == "" || raw == "-" || raw == "N/A" {
				continue
			}
			n, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || n < 0 {
				return nil, fmt.Errorf("invalid nonnegative integer in %s", name)
			}
			*dest = &n
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		return nil, errors.New("CSV contains no data rows")
	}
	return rows, nil
}
