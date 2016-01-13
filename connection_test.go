package stringutil

import "testing"
import "os"
// import "github.com/golang/protobuf/proto"
// import "bytes"
// import "log"

func TestConnection(t *testing.T) {
	s := Session{}
	s.StartConnection()
	s.Login()
	s.Run()

	username := os.Getenv("SPOT_USERNAME")
	sController := SetupController(&s, username, "7288edd0fc3ffcbe93a0cf06e3568e28521687bc")
	sController.run()



}

// func TestCreateHello(t *testing.T) {
// 	public := []byte{
// 		38, 240, 157, 80, 168, 248, 161, 180, 252, 103, 26, 99, 14, 113, 0, 63, 224, 56, 125, 135, 126, 53, 232, 82, 88, 73, 47, 192, 161, 1, 185, 20, 35, 217, 224, 27, 187, 163, 244, 154, 128, 158, 36, 71, 27, 171, 13, 94, 43, 53, 81, 25, 253, 171, 40, 233, 237, 255, 38, 41, 43, 142, 92, 92, 46, 109, 8, 87, 127, 192, 171, 70, 124, 182, 201, 118, 250, 228, 156, 160, 177, 69, 168, 12, 206, 119, 13, 9, 149, 82, 184, 131, 131, 162, 172, 55,
// 	}
// 	want := []byte{
// 		82, 13, 80, 5, 240, 1, 2, 192, 2, 128, 128, 128, 128, 128, 33, 240, 1, 0, 146, 3, 103, 82, 101, 82, 96, 38, 240, 157, 80, 168, 248, 161, 180, 252, 103, 26, 99, 14, 113, 0, 63, 224, 56, 125, 135, 126, 53, 232, 82, 88, 73, 47, 192, 161, 1, 185, 20, 35, 217, 224, 27, 187, 163, 244, 154, 128, 158, 36, 71, 27, 171, 13, 94, 43, 53, 81, 25, 253, 171, 40, 233, 237, 255, 38, 41, 43, 142, 92, 92, 46, 109, 8, 87, 127, 192, 171, 70, 124, 182, 201, 118, 250, 228, 156, 160, 177, 69, 168, 12, 206, 119, 13, 9, 149, 82, 184, 131, 131, 162, 172, 55, 160, 1, 1, 226, 3, 2, 0, 0, 130, 5, 2, 8, 1,
// 	}

// 	hello := CreateHello(public)
//     res, err := proto.Marshal(hello)
//     if err != nil {
//         log.Fatal("marshaling error: ", err)
//     }

//   	if !bytes.Equal(res, want) {
// 		t.Errorf("result does not match, %v %v", want, res)
// 	}
// }

// func TestMakePacketPrefix(t *testing.T) {
// 	makePacketPrefix([]byte('prefix'),[]byte('some cool'))
// }
