package common

import (
	"testing"
)

func TestSuggestedActions_RequiresReboot(t *testing.T) {
	tests := []struct {
		name string
		sa   SuggestedActions
		want bool
	}{
		{
			name: "nil",
			sa:   SuggestedActions{},
			want: false,
		},
		{
			name: "requires reboot",
			sa: SuggestedActions{
				RepairActions: []RepairActionType{RepairActionTypeRebootSystem},
			},
			want: true,
		},
		{
			name: "requires reboot and repair hardware",
			sa: SuggestedActions{
				RepairActions: []RepairActionType{RepairActionTypeRebootSystem, RepairActionTypeRepairHardware},
			},
			want: true,
		},
		{
			name: "does not require reboot",
			sa: SuggestedActions{
				RepairActions: []RepairActionType{RepairActionTypeRepairHardware},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sa.RequiresReboot(); got != tt.want {
				t.Errorf("SuggestedActions.RequiresReboot() = %v, want %v", got, tt.want)
			}
		})
	}
}