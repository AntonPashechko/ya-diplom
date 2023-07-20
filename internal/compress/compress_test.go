package compress

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGzipCompress(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"compress test",
			args{
				[]byte("Hello"),
			},
			"\x1f\x8b\b\x00\x00\x00\x00\x00\x00\xff\xf2H\xcd\xc9\xc9\a\x04\x00\x00\xff\xff\x82\x89\xd1\xf7\x05\x00\x00\x00",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GzipCompress(tt.args.data)
			assert.NoError(t, err)

			assert.Equal(t, tt.want, string(got))
		})
	}
}
