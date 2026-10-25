// Package clock は OS の時計から現在時刻を取得する実装を提供します。
package clock

import (
	"time"

	"github.com/kenyamaneko/overload-party-card/internal/port"
)

var _ port.Clock = (*SystemClock)(nil)

// SystemClock は OS の時計を使用した Clock の実装です。
type SystemClock struct{}

// NewSystemClock は SystemClock を生成します。
func NewSystemClock() *SystemClock {
	return &SystemClock{}
}

// Now は OS の時計の現在時刻を返します。
func (c *SystemClock) Now() time.Time {
	return time.Now()
}
