package stringutil

import "testing"
import "bytes"

func setupShannon() ShannonStream{
	shannon := ShannonStream{}
	shannon.SetSendKey([]uint8("cool key 123"))
	shannon.SetRecvKey([]uint8("cool key 123"))
	return shannon
}

func TestShnKey(t *testing.T) {
	shannon := setupShannon()
	res := shannon.Encrypt("hi123")

	want := []uint8{102,155,110,118,118}
	if !bytes.Equal(res, want) {
		t.Errorf("result does not match, %v %v", want, res)
	}
}

func TestFinishSend(t *testing.T) {
	shannon := setupShannon()
	res := shannon.Encrypt("hi123")
	res = append(res, shannon.FinishSend()...)

	want := []uint8{102, 155, 110, 118, 118, 231, 12, 121, 114}
	if !bytes.Equal(res, want) {
		t.Errorf("result does not match, %v %v", want, res)
	}
}

func TestDecode(t *testing.T) {
	shannon := setupShannon()
	res := shannon.Encrypt("hi123")
	res = shannon.Decrypt(res)

	if !bytes.Equal(res, []byte("hi123")) {
		t.Errorf("result does not match, %v %v", res, []byte("hi123"))
	}

}

// func TestReverse(t *testing.T) {
// 	cases := []struct {
// 		in, want string
// 	}{
// 		{"Hello, world", "dlrow ,olleH"},
// 		{"Hello, 世界", "界世 ,olleH"},
// 		{"", ""},
// 	}
// 	for _, c := range cases {
// 		got := Reverse(c.in)
// 		if got != c.want {
// 			t.Errorf("Reverse(%q) == %q, want %q", c.in, got, c.want)
// 		}
// 	}
// }