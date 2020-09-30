package utility

import "github.com/ssgo/u"

func EncodeNumber(num uint64, offset uint64) string {
	return string(u.EncodeInt(num - 9567783217 + offset))
}
