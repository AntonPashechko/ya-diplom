package utils

import (
	"fmt"
	"strconv"
)

/*Просто переводит строку в int64 с проверкой ошибки*/
func StrToInt64(str string) (int64, error) {
	if str != "" {
		return strconv.ParseInt(str, 10, 64)
	}
	return 0, nil
}

/*Просто переводит строку в float64 с проверкой ошибки*/
func StrToFloat64(str string) (float64, error) {
	if str != "" {
		return strconv.ParseFloat(str, 64)
	}
	return 0, nil
}

/*Просто переводит float64 в строку*/
func Int64ToStr(i64 int64) string {
	return fmt.Sprintf("%d", i64)
}

/*Просто переводит float64 в строку*/
func Float64ToStr(f64 float64) string {
	return strconv.FormatFloat(f64, 'f', -1, 64)
}

func DeepCopyMap[K comparable, V any](src map[K]V) map[K]V {
	dest := make(map[K]V, len(src))

	for k, v := range src {
		dest[k] = v
	}

	return dest
}

// Valid check number is valid or not based on Luhn algorithm
func ValidLuhn(number int64) bool {
	return (number%10+checksum(number/10))%10 == 0
}

func checksum(number int64) int64 {
	var luhn int64

	for i := 0; number > 0; i++ {
		cur := number % 10

		if i%2 == 0 {
			cur = cur * 2
			if cur > 9 {
				cur = cur%10 + cur/10
			}
		}

		luhn += cur
		number = number / 10
	}
	return luhn % 10
}
