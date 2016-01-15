package spotcontrol

import (
	"strings"
	"math/big"
)

func Convert62 (id string) []byte{
	digits := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	base := big.NewInt(62)

	n := &big.Int{}
	for _, c := range []byte(id) {
		d := big.NewInt(int64(strings.IndexByte(digits, c)))
		n = n.Mul(n, base)
		n = n.Add(n, d)
	}

	return n.Bytes()
}