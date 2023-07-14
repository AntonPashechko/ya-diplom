package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitialize(t *testing.T) {
	type args struct {
		level string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"init_loger_test",
			args{
				"info",
			},
			false,
		},
		{
			"bad_level",
			args{
				"info1",
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Initialize(tt.args.level)
			if !tt.wantErr {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestInfo(t *testing.T) {
	type args struct {
		msg string
		opt []any
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"log_info",
			args{
				msg: "Test log info",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Info(tt.args.msg, tt.args.opt...)
		})
	}
}

func TestError(t *testing.T) {
	type args struct {
		msg string
		opt []any
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"log_error",
			args{
				msg: "Test log error",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Error(tt.args.msg, tt.args.opt...)
		})
	}
}
