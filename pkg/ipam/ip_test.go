package ipam

import (
	"reflect"
	"testing"
)

func TestGetIPRangeList(t *testing.T) {
	type args struct {
		ips []IP
	}
	tests := []struct {
		name string
		args args
		want []*IPRange
	}{
		{
			name: "testcase1",
			args: args{
				ips: []IP{"192.168.10.11", "192.168.1.10", "192.168.10.10", "192.168.10.9"},
			},
			want: []*IPRange{
				{Start: "192.168.1.10", End: "192.168.1.10"},
				{Start: "192.168.10.9", End: "192.168.10.11"},
			},
		},
		{
			name: "testcase2",
			args: args{
				ips: []IP{},
			},
			want: []*IPRange{},
		},
		{
			name: "testcase3",
			args: args{
				ips: []IP{"10.0.0.0"},
			},
			want: []*IPRange{{Start: "10.0.0.0", End: "10.0.0.0"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetIPRangeList(tt.args.ips); !reflect.DeepEqual(got, tt.want) {
				for _, ips := range got {
					t.Errorf("got: %v", ips)
				}
				for _, ips := range tt.want {
					t.Errorf("want: %v", ips)
				}
			}
		})
	}
}

func TestIP_Add(t *testing.T) {
	type args struct {
		n uint32
	}
	tests := []struct {
		name string
		ip   IP
		args args
		want IP
	}{
		{
			name: "testcace1",
			ip:   "192.168.50.255",
			args: args{
				n: 1,
			},
			want: "192.168.51.0",
		},
		{
			name: "testcace2",
			ip:   "192.168.50.255",
			args: args{
				n: 257,
			},
			want: "192.168.52.0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ip.Add(tt.args.n); got != tt.want {
				t.Errorf("IP.Add() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIP_Sub(t *testing.T) {
	type args struct {
		n uint32
	}
	tests := []struct {
		name string
		ip   IP
		args args
		want IP
	}{
		{
			name: "testcace1",
			ip:   "192.168.50.255",
			args: args{
				n: 1,
			},
			want: "192.168.50.254",
		},
		{
			name: "testcace2",
			ip:   "192.168.50.0",
			args: args{
				n: 1,
			},
			want: "192.168.49.255",
		},
		{
			name: "testcace3",
			ip:   "192.168.50.255",
			args: args{
				n: 257,
			},
			want: "192.168.49.254",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ip.Sub(tt.args.n); got != tt.want {
				t.Errorf("IP.Sub() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIP_LessThan(t *testing.T) {
	type args struct {
		b IP
	}
	tests := []struct {
		name string
		ip   IP
		args args
		want bool
	}{
		{
			name: "testcase1",
			ip:   "192.168.50.1",
			args: args{
				b: "192.168.51.0",
			},
			want: true,
		},
		{
			name: "testcase2",
			ip:   "192.168.50.1",
			args: args{
				b: "192.168.50.1",
			},
			want: false,
		},
		{
			name: "testcase3",
			ip:   "192.168.50.1",
			args: args{
				b: "192.168.0.1",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ip.LessThan(tt.args.b); got != tt.want {
				t.Errorf("IP.LessThan() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIP_GreaterThan(t *testing.T) {
	type args struct {
		b IP
	}
	tests := []struct {
		name string
		ip   IP
		args args
		want bool
	}{
		{
			name: "testcase1",
			ip:   "192.168.50.1",
			args: args{
				b: "192.168.51.0",
			},
			want: false,
		},
		{
			name: "testcase2",
			ip:   "192.168.50.1",
			args: args{
				b: "192.168.50.1",
			},
			want: false,
		},
		{
			name: "testcase3",
			ip:   "192.168.50.1",
			args: args{
				b: "192.168.0.1",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ip.GreaterThan(tt.args.b); got != tt.want {
				t.Errorf("IP.GreaterThan() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIP_Equal(t *testing.T) {
	type args struct {
		b IP
	}
	tests := []struct {
		name string
		ip   IP
		args args
		want bool
	}{
		{
			name: "testcase1",
			ip:   "192.168.50.1",
			args: args{
				b: "192.168.51.0",
			},
			want: false,
		},
		{
			name: "testcase2",
			ip:   "192.168.50.1",
			args: args{
				b: "192.168.50.1",
			},
			want: true,
		},
		{
			name: "testcase3",
			ip:   "192.168.50.1",
			args: args{
				b: "192.168.0.1",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ip.Equal(tt.args.b); got != tt.want {
				t.Errorf("IP.Equal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIPRange_Contains(t *testing.T) {
	type fields struct {
		Start IP
		End   IP
	}
	type args struct {
		ip IP
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "testcase1",
			fields: fields{
				Start: "192.168.50.1",
				End:   "192.168.50.1",
			},
			args: args{
				ip: "192.168.50.1",
			},
			want: true,
		},
		{
			name: "testcase2",
			fields: fields{
				Start: "192.168.50.1",
				End:   "192.168.50.1",
			},
			args: args{
				ip: "192.168.50.2",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &IPRange{
				Start: tt.fields.Start,
				End:   tt.fields.End,
			}
			if got := r.Contains(tt.args.ip); got != tt.want {
				t.Errorf("IPRange.Contains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIPRangeList_Contains(t *testing.T) {
	type args struct {
		ip IP
	}
	tests := []struct {
		name string
		iprl IPRangeList
		args args
		want bool
	}{
		{
			name: "testcase1",
			iprl: IPRangeList(
				[]*IPRange{
					{
						Start: "192.168.50.1",
						End:   "192.168.50.1",
					},
					{
						Start: "192.168.50.49",
						End:   "192.168.50.50",
					},
				},
			),
			args: args{
				ip: "192.168.50.2",
			},
			want: false,
		},
		{
			name: "testcase2",
			iprl: IPRangeList(
				[]*IPRange{
					{
						Start: "192.168.50.1",
						End:   "192.168.50.1",
					},
					{
						Start: "192.168.50.49",
						End:   "192.168.50.50",
					},
				},
			),
			args: args{
				ip: "192.168.50.1",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.iprl.Contains(tt.args.ip); got != tt.want {
				t.Errorf("IPRangeList.Contains() = %v, want %v", got, tt.want)
			}
		})
	}
}
