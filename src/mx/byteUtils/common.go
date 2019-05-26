package byteUtils

import (
	"strings"
	"encoding/hex"
	"bufio"
)

/**
 * 中间带空格的
 * 68 56 00 56 00 68 4B 00 12 51 BC 02 0D E9 40 04 01 14 22 11 17 19 16 02 04 23 00 5D 16
 */
func HexToSlice(hexStr string) ([]byte, error) {
	hexStr = strings.TrimSpace(hexStr);
	var slice = make([]byte, 0, 20)
	for _, byteHex := range strings.Split(hexStr, " ") {
		slice0, err := hex.DecodeString(byteHex);
		if err != nil {
			return nil, err
		}
		slice = append(slice, slice0[0]);
	}

	return slice, nil
}

/*
数组转成中间带空格的hex传值
 */
func SliceToHex(slice []byte) string {
	sb := make([]string, 0, 20)
	for _, by := range slice {
		str := hex.EncodeToString([]byte{by})
		sb = append(sb, strings.ToUpper(str));
	}
	return strings.Join(sb, " ")
}


/**
 * 直到从reader中读取n个字节才返回,否则tcp一直阻塞
 */
func ReadN(n int, reader *bufio.Reader) ([]byte, error) {
	for {
		buf, err := reader.Peek(n)

		if err == nil {
			reader.Discard(n)
			return buf, nil
		}

		if err != bufio.ErrBufferFull {
			return nil, err;
		}
	}
}