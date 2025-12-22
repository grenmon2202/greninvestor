package charts

import "time"

type Candle struct {
	T time.Time
	O float64
	H float64
	L float64
	C float64
	V float64
}
