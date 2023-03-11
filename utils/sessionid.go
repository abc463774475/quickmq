package utils

import (
	"fmt"
	"time"

	_ "github.com/abc463774475/snowflake"
)

func GetSessionIDByTimer() string {
	curTime := time.Now()
	t := "20060102150405"

	str := curTime.Format(t) + fmt.Sprintf("%v%v", curTime.Nanosecond(), GetRandInt32())
	return str
}

func GetSessionIDBySnowflake() int64 {
	return 0
}
