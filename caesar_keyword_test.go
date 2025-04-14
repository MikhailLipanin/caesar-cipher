package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEncrypt(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		keyword string
		shift   int

		want    assert.ValueAssertionFunc
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "happy path",
			text:    "SENd MORE MONEY",
			keyword: "DIPLO_MAT",
			shift:   5,
			want: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				return assert.Equal(t, i.(string), "HZBy TCGZ TCBZS")
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Encrypt(tt.text, tt.keyword, tt.shift)
			tt.wantErr(t, err)
			tt.want(t, got)
		})
	}
}

func TestDecrypt(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		keyword string
		shift   int

		want    assert.ValueAssertionFunc
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "happy path",
			text:    "HZBy TCGZ TCBZS",
			keyword: "DIPLO_MAT",
			shift:   5,
			want: func(t assert.TestingT, i interface{}, i2 ...interface{}) bool {
				return assert.Equal(t, i.(string), "SENd MORE MONEY")
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Decrypt(tt.text, tt.keyword, tt.shift)
			tt.wantErr(t, err)
			tt.want(t, got)
		})
	}
}
