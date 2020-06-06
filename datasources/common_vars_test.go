package datasources

import (
	"fmt"
	"testing"
	"time"
)

func TestTimeStr(t *testing.T) {
	expected := map[int]string{
		0:        "just now",
		1:        "1s",
		10:       "10s",
		59:       "59s",
		60:       "1m",
		61:       "1m1s",
		119:      "1m59s",
		120:      "2m",
		38970000: "1yr2mo",
	}
	for i := 0; i < 1000; i++ {
		for k, v := range expected {
			d, _ := time.ParseDuration(fmt.Sprintf("%ds", k))
			actual := timeStr(d, 2, true)
			if actual != v {
				t.Errorf("%d seconds: got %s, expected %s", k, actual, v)
			}
		}
	}
}

func BenchmarkTimeStr(b *testing.B) {
	// 1 year 2 months 3 weeks 4 days 5 hours 6 minutes 7 seconds
	d, _ := time.ParseDuration("38970000s")
	for i := 0; i < b.N; i++ {
		timeStr(d, 2, true)
	}
}
