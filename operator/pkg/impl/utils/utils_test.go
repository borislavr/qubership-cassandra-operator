package utils

import (
	"reflect"
	"testing"

	"github.com/Netcracker/qubership-cassandra-operator/api/v1alpha1"
)

func TestFilterDC(t *testing.T) {
	type args struct {
		input  []*v1alpha1.DataCenter
		filter func(dc *v1alpha1.DataCenter) bool
	}
	dcsAll := []*v1alpha1.DataCenter{
		{
			Deploy:   true,
			Name:     "dc1",
			Replicas: 3,
		},
		{
			Deploy:   true,
			Name:     "dc2",
			Replicas: 3,
		},
	}

	dcsOne := []*v1alpha1.DataCenter{
		{
			Deploy:   true,
			Name:     "dc1",
			Replicas: 3,
		},
		{
			Deploy:   false,
			Name:     "dc2",
			Replicas: 3,
		},
	}

	tests := []struct {
		name string
		args args
		want []*v1alpha1.DataCenter
	}{
		{
			name: "Test All DC deployed",
			args: args{
				input:  dcsAll,
				filter: func(dc *v1alpha1.DataCenter) bool { return dc.Deploy },
			},
			want: dcsAll,
		},
		{
			name: "Test One DC deployed",
			args: args{
				input:  dcsOne,
				filter: func(dc *v1alpha1.DataCenter) bool { return dc.Deploy },
			},
			want: []*v1alpha1.DataCenter{
				{
					Deploy:   true,
					Name:     "dc1",
					Replicas: 3,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// mass := NewTypedStream(dcsAll, &v13.DataCenter{}).Slice()
			// nn := make([]*v13.DataCenter, len(mass))
			// for i, mas := range mass {
			// 	nn[i] = mas.(*v13.DataCenter)
			// }
			if got := FilterDC(tt.args.input, tt.args.filter); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FilterDC() = %v, want %v", got, tt.want)
			}
		})
	}
}
