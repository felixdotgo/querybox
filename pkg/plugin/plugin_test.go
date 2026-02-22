package plugin

import "testing"

func TestFormatSQLValue(t *testing.T) {
    tests := []struct {
        name string
        input interface{}
        want string
    }{
        {"nil", nil, ""},
        {"string", "foo", "foo"},
        {"int", 42, "42"},
        {"bool", true, "true"},
        {"float", 3.14, "3.14"},
        {"bytes", []byte("hello"), "hello"},
        {"ascii bytes", []byte{0x41, 0x42, 0x43}, "ABC"},
        {"non-utf8 bytes", []byte{0xff, 0xfe}, "0xfffe"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := FormatSQLValue(tt.input)
            if got != tt.want {
                t.Errorf("FormatSQLValue(%v) = %q; want %q", tt.input, got, tt.want)
            }
        })
    }
}
