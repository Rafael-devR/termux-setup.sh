package sys

import (
	"os"
	"runtime"
	"time"
	_ "time/tzdata" // base de fusos embutida: o Android não tem /usr/share/zoneinfo
)

// FixTimezone acerta o fuso horário nos binários GOOS=linux (32 bits e
// x86): o Android não tem /etc/localtime, então o Go cairia em UTC e os
// logs sairiam com a hora errada. O fuso vem de "getprop persist.sys.timezone".
func FixTimezone() {
	if runtime.GOOS == "android" || os.Getenv("TZ") != "" || time.Local.String() != "UTC" {
		return
	}
	if tz := getprop("persist.sys.timezone"); tz != "?" {
		if loc, err := time.LoadLocation(tz); err == nil {
			time.Local = loc
		}
	}
}
