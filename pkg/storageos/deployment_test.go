package storageos

import "testing"

func Test_isV2image(t *testing.T) {

	t.Parallel()

	tests := []struct {
		image string
		want  bool
	}{
		{image: "storageos/node", want: false},
		{image: "storageos/node:1.0.0", want: false},
		{image: "storageos/node:1.2.0-alpha1", want: false},
		{image: "storageos/node:7c46250197bf", want: false},
		{image: "storageos/node:2.0.0", want: true},
		{image: "storageos/node:2.0.0-alpha1", want: true},
		{image: "storageos/node:c2-7c46250197bf", want: true},
		{image: "myregistryhost:5000/storageos/node:1.0.0", want: false},
		{image: "myregistryhost:5000/storageos/node:2.0.0", want: true},
		{image: "invalidscheme://myregistryhost:5000/storageos/node:2.0.0", want: true},
		{image: "2.0.0", want: false},
	}
	for _, tt := range tests {
		var tt = tt
		t.Run(tt.image, func(t *testing.T) {
			t.Parallel()
			if got := isV2image(tt.image); got != tt.want {
				t.Errorf("isV2image(%s) = %v, want %v", tt.image, got, tt.want)
			}
		})
	}
}
