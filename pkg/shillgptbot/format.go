package shillgptbot

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/spf13/viper"
)

const (
	// DateLayoutISO - date format yyyy-mm-dd
	DateLayoutISO = "2006-01-02"

	// DateTimeLayoutISO - date time format yyyy-mm-dd hh:ii:ss
	DateTimeLayoutISO = "2006-01-02 15:04:05"

	// ShortDateLayout - date format yy-mm-dd
	ShortDateLayout = "06-01-02"

	// ShortDateTimeLayout - date time format yy-mm-dd hh:ii:ss
	ShortDateTimeLayout = "06-01-02 15:04:05"

	// MicroDateTimeLayout
	MicroDateTimeLayout = "2006-01-02 15:04:05.000000"
)

// convert millisecond timestamp to time.Time
func MilliToTime(timestamp int64) time.Time {
	return time.Unix(0, timestamp*int64(time.Millisecond))
}

func StrToFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func ApplyPrecision(amount float64, size string, roundUp bool) float64 {
	sizeAsFloat, _ := strconv.ParseFloat(size, 64)
	if sizeAsFloat >= 1 {
		return math.Floor(amount / sizeAsFloat)
	}

	// get precision
	parts := strings.Split(size, "1")
	precision := len(parts[0]) - 1
	//formatter := fmt.Sprintf("%%0.%df", precision)

	// apply precision and round down
	pow := math.Pow(10, float64(precision))
	multipled := pow * amount

	var rounded float64
	if roundUp {
		rounded = math.Ceil(multipled)
	} else {
		rounded = math.Floor(multipled)
	}
	return rounded / pow
}

func TimeToMongoDate(t time.Time) string {
	rfc3339MilliLayout := "2006-01-02T15:04:05.999Z07:00" // layout defined with Go reference time
	return t.Format(rfc3339MilliLayout)
}

func DateToMongoDate(date string) time.Time {
	rfc3339MilliLayout := "2006-01-02T15:04:05.999Z07:00" // layout defined with Go reference time
	parsedDate, _ := time.Parse(rfc3339MilliLayout, date)

	return parsedDate
}

func TimestampToIsoDate(timestamp int64) string {
	isoLayout := "2006-01-02T15:04:05"
	return MilliToTime(timestamp).Format(isoLayout)
}

func KlineRedGreen(green bool) string {
	colour := "green"
	if !green {
		colour = "red"
	}
	return colour
}

func Lcfirst(str string) string {
	for _, v := range str {
		u := string(unicode.ToLower(v))
		return u + str[len(u):]
	}
	return ""
}

func FormatFloatString(s string, precision int) string {
	f := StrToFloat(s)
	return FormatFloat(f, precision)
}

func FormatFloat(f float64, precision int) string {
	fullFloat := fmt.Sprintf("%v", f)
	parts := strings.Split(fullFloat, ".")

	if len(parts) == 2 {
		for _, c := range parts[1] {
			if string(c) != "0" {
				break
			}
			precision++
		}
	}

	return fmt.Sprintf("%0.*f", precision, f)
}

func ToLocalTime(nonLocalTime time.Time) time.Time {
	loc, _ := time.LoadLocation(viper.GetString("timezone"))
	return nonLocalTime.In(loc)
}

func LocalToUTCTime(localTime time.Time) time.Time {
	loc, _ := time.LoadLocation(viper.GetString("timezone"))
	_, offset := time.Now().In(loc).Zone()
	diff := time.Duration(offset) * time.Second

	return localTime.Add(-diff).UTC()
}

func PercentDiff(value1 float64, value2 float64) float64 {
	return ((value1 - value2) / value2) * 100
}
