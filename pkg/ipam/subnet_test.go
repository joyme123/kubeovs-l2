package ipam

import (
	"regexp"
	"testing"
)

func newSubnetTest() *Subnet {
	n, _ := NewSubnet("subnet1",
		"192.168.1.0/24",
		[]IP{"192.168.1.1", "192.168.1.2", "192.168.1.127"})
	return n
}

func TestSubnet_GetRandomMac(t *testing.T) {
	type args struct {
		pod string
	}
	tests := []struct {
		name   string
		subnet *Subnet
		args   args
		want   string
	}{
		{
			name:   "testcase1",
			subnet: newSubnetTest(),
			args: args{
				pod: "pod1",
			},
			want: "([0-9A-F]){2}:([0-9A-F]){2}:([0-9A-F]){2}:([0-9A-F]){2}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subnet := tt.subnet
			got := subnet.GetRandomMac(tt.args.pod)

			matched, err := regexp.MatchString(tt.want, got)
			if err != nil {
				t.Errorf("got err: %v", err)
			}

			if !matched {
				t.Error("does't match, got:", got)
			}
		})
	}
}
