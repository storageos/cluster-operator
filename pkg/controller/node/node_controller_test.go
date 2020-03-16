package node

import (
	"reflect"
	"testing"
)

func TestUpdateLabels(t *testing.T) {
	tests := []struct {
		name       string
		arg1, arg2 map[string]string
		ret        bool
	}{
		{
			name: "empty args",
			arg1: map[string]string{},
			arg2: map[string]string{},
			ret:  false,
		},
		{
			name: "uninitialized args",
			arg1: nil,
			arg2: nil,
			ret:  false,
		},
		{
			// Case of StorageOS v1 when node labels are empty.
			name: "arg1 uninitialized",
			arg1: nil,
			arg2: map[string]string{},
			ret:  false,
		},
		{
			name: "new items in arg2",
			arg1: map[string]string{"foo1": "bar1"},
			arg2: map[string]string{"foo1": "bar1", "foo2": "bar2"},
			ret:  true,
		},
		{
			name: "new values for same item in arg2",
			arg1: map[string]string{"foo1": "bar1", "foo2": "bar2"},
			arg2: map[string]string{"foo1": "bar1", "foo2": "bar3"},
			ret:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ret := updateLabels(tt.arg1, tt.arg2)
			if tt.ret != ret {
				t.Errorf("unexpected updateLabels result:\n\t(WNT) %t\n\t(GOT) %t", tt.ret, ret)
			}

			// Check if arg1 and arg2 are equal after the update.

			// DeepEqual empty maps results in false. Compare values only when
			// there are any elements in the map.
			if len(tt.arg2) > 0 {
				if !reflect.DeepEqual(tt.arg1, tt.arg2) {
					t.Errorf("expected the labels to be equal after update, arg1: %v, arg2: %v", tt.arg1, tt.arg2)
				}
			} else {
				// If arg2 is empty, check if arg1 is also empty after the update.
				if len(tt.arg1) != len(tt.arg2) {
					t.Errorf("expected the labels to be equal after update, arg1: %v, arg2: %v", tt.arg1, tt.arg2)
				}
			}
		})
	}
}
