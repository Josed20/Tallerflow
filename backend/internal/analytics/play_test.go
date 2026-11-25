package analytics

import (
	"encoding/binary"
	"strings"
	"testing"
	"time"
	"unicode/utf16"

	"github.com/stretchr/testify/require"
)

var playNow = time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)

const playCSV = "Date,Package Name,App Version Code,Daily Device Installs,Daily User Installs,Daily Device Uninstalls\n2026-11-23,pe.tallerflow.app,3,12,,0\n"

func TestPlayCSVPreservesMissingVersusZeroAndNumericVersion(t *testing.T) {
	rows, err := ParsePlayCSV([]byte(playCSV), playNow)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "3", rows[0].Version)
	require.Equal(t, "app_version", rows[0].Dimension)
	require.EqualValues(t, 12, *rows[0].DeviceInstalls)
	require.Nil(t, rows[0].UserInstalls)
	require.EqualValues(t, 0, *rows[0].Uninstalls)
	require.True(t, rows[0].HasDeviceInstalls)
	require.True(t, rows[0].HasUserInstalls, "a present blank column is an explicit unavailable value")
	require.True(t, rows[0].HasUninstalls)
}

func TestPlayCSVTracksOmittedColumnsSeparatelyFromExplicitBlanks(t *testing.T) {
	for _, raw := range []string{"", "-", "N/A"} {
		csv := "Date,Package Name,Daily Device Installs,Daily User Installs\n2026-11-23,pe.tallerflow.app,7," + raw + "\n"
		rows, err := ParsePlayCSV([]byte(csv), playNow)
		require.NoError(t, err)
		require.True(t, rows[0].HasDeviceInstalls)
		require.True(t, rows[0].HasUserInstalls)
		require.Nil(t, rows[0].UserInstalls, "present blank explicitly clears the stored value")
		require.False(t, rows[0].HasUninstalls, "an omitted column must preserve its stored value")
		require.Nil(t, rows[0].Uninstalls)
	}
}

func TestPlayCSVCanonicalizesVersionBeforeDuplicateDetection(t *testing.T) {
	rows, err := ParsePlayCSV([]byte(strings.Replace(playCSV, ",3,", ",03,", 1)), playNow)
	require.NoError(t, err)
	require.Equal(t, "3", rows[0].Version)
	_, err = ParsePlayCSV([]byte(playCSV+"2026-11-23,pe.tallerflow.app,03,5,2,1\n"), playNow)
	require.ErrorContains(t, err, "duplicate date/version")
}

func TestPlayCSVSupportsOfficialUTF16BothByteOrders(t *testing.T) {
	for _, order := range []binary.ByteOrder{binary.LittleEndian, binary.BigEndian} {
		units := append([]uint16{0xfeff}, utf16.Encode([]rune(playCSV))...)
		data := make([]byte, len(units)*2)
		for i, v := range units {
			order.PutUint16(data[i*2:], v)
		}
		rows, err := ParsePlayCSV(data, playNow)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		require.EqualValues(t, 12, *rows[0].DeviceInstalls)
	}
}

func TestPlayCSVRejectsAmbiguityAndBadCounts(t *testing.T) {
	for name, data := range map[string]string{
		"wrong package":        "Date,Package Name,Daily Device Installs\n2026-11-23,other.app,1",
		"country dimension":    "Date,Package Name,Country,Daily Device Installs\n2026-11-23,pe.tallerflow.app,PE,1",
		"duplicate key":        playCSV + "2026-11-23,pe.tallerflow.app,3,1,1,1\n",
		"negative":             "Date,Package Name,Daily Device Installs\n2026-11-23,pe.tallerflow.app,-1",
		"future":               "Date,Package Name,Daily Device Installs\n2026-11-25,pe.tallerflow.app,1",
		"expired":              "Date,Package Name,Daily Device Installs\n2023-01-01,pe.tallerflow.app,1",
		"name instead of code": "Date,Package Name,App Version Code,Daily Device Installs\n2026-11-23,pe.tallerflow.app,1.1.0,1",
		"duplicate column":     "Date,Package Name,Daily Device Installs,Daily Device Installs\n2026-11-23,pe.tallerflow.app,1,1",
		"empty":                "Date,Package Name,Daily Device Installs\n",
		"overflow":             "Date,Package Name,Daily Device Installs\n2026-11-23,pe.tallerflow.app,9223372036854775808",
	} {
		t.Run(name, func(t *testing.T) { _, err := ParsePlayCSV([]byte(data), playNow); require.Error(t, err) })
	}
}
