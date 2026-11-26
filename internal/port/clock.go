package port

import "time"

// Clock は現在時刻を提供するインターフェースです。
type Clock interface {
	// Now は現在時刻を返します。
	Now() time.Time
}
