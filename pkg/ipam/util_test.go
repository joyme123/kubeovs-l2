package ipam

import (
	"reflect"
	"regexp"
	"testing"
)

func TestBytesToUint32(t *testing.T) {
	type args struct {
		bs []byte
	}
	tests := []struct {
		name string
		args args
		want uint32
	}{
		{
			name: "testcase1",
			args: args{
				bs: []byte{192, 168, 33, 0},
			},
			want: 3232243968,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BytesToUint32(tt.args.bs); got != tt.want {
				t.Errorf("BytesToUint32() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUint32ToBytes(t *testing.T) {
	type args struct {
		n uint32
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "testcase1",
			args: args{
				n: 3232243968,
			},
			want: []byte{192, 168, 33, 0},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Uint32ToBytes(tt.args.n); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Uint32ToBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetFirstIP(t *testing.T) {
	type args struct {
		cidr string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "testcase1",
			args: args{
				cidr: "192.168.33.0/25",
			},
			want: "192.168.33.1",
		},
		{
			name: "testcase2",
			args: args{
				cidr: "192.168.33.127/25",
			},
			want: "192.168.33.1",
		},
		{
			name: "testcase3",
			args: args{
				cidr: "192.168.33.128/25",
			},
			want: "192.168.33.129",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetFirstIP(tt.args.cidr); got != tt.want {
				t.Errorf("GetFirstIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetLastIP(t *testing.T) {
	type args struct {
		cidr string
	}
	tests := []struct {
		name string
		args args
		want string
	}{

		{
			name: "testcase1",
			args: args{
				cidr: "192.168.33.0/25",
			},
			want: "192.168.33.126",
		},
		{
			name: "testcase2",
			args: args{
				cidr: "192.168.33.127/25",
			},
			want: "192.168.33.126",
		},
		{
			name: "testcase3",
			args: args{
				cidr: "192.168.33.128/25",
			},
			want: "192.168.33.254",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetLastIP(tt.args.cidr); got != tt.want {
				t.Errorf("GetLastIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateMac(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "testcase1",
			want: "([0-9A-F]){2}:([0-9A-F]){2}:([0-9A-F]){2}:([0-9A-F]){2}",
		},
		{
			name: "testcase2",
			want: "([0-9A-F]){2}:([0-9A-F]){2}:([0-9A-F]){2}:([0-9A-F]){2}",
		},
	}
	pre := ""
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateMac()
			if got == pre {
				t.Errorf("mac is not random, pre: %v, got: %v", pre, got)
			}
			pre = got

			match, err := regexp.MatchString(tt.want, got)

			if err != nil {
				t.Errorf("got err: %v", err)
			}

			if !match {
				t.Error("does't match, got:", got)
			}
		})
	}
}

func BenchmarkGenerateMac(b *testing.B) {
	GenerateMac()
}
