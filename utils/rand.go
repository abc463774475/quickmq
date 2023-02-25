package utils

import "math/rand"

func GetRandInt32() int32 {
	return rand.Int31()
}

func GetRandInt8() int8 {
	return int8(rand.Int31n(256))
}
