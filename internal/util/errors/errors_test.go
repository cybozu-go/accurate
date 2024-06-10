package errors

import (
	"errors"
	"testing"
)

func TestIgnore(t *testing.T) {
	err := errors.New("error")

	type args struct {
		err error
		is  []ErrorIs
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "ignored", args: args{err: err, is: []ErrorIs{IsErrorAlways}}},
		{name: "not ignored", args: args{err: err, is: []ErrorIs{IsErrorNever}}, wantErr: true},
		{name: "ignored by first", args: args{err: err, is: []ErrorIs{IsErrorAlways, IsErrorNever}}},
		{name: "ignored by second", args: args{err: err, is: []ErrorIs{IsErrorNever, IsErrorAlways}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Ignore(tt.args.err, tt.args.is...); (err != nil) != tt.wantErr {
				t.Errorf("IgnoreAny() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

//goland:noinspection GoUnusedParameter
func IsErrorAlways(err error) bool {
	return true
}

//goland:noinspection GoUnusedParameter
func IsErrorNever(err error) bool {
	return false
}
