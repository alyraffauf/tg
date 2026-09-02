package spindle

import "testing"

func FuzzDecodeLogEvent(f *testing.F) {
	f.Add([]byte{0xa1, 0x61, 0x74, 0x65, 0x23, 0x64, 0x61, 0x74, 0x61, 0xa0})
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > maxPipelineLogMessageBytes {
			t.Skip()
		}
		_, _ = decodeLogEvent(data)
	})
}
