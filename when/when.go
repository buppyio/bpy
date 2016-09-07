package when

import (
	"strings"
	"time"
)

func Parse(whenString string) (time.Time, error) {
	if strings.HasSuffix(whenString, " ago") {
		d, err := time.ParseDuration(whenString[0 : len(whenString)-4])
		if err != nil {
			return time.Time{}, err
		}
		return time.Now().Add(-d), nil
	} else {
		return time.Parse("", whenString)
	}
}
