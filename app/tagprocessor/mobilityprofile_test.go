package tagprocessor

import "testing"

func TestNewMobilityProfile(t *testing.T) {
	// check that default is asset tracking
	mp := NewMobilityProfile()
	if mp.M >= 0.0 {
		t.Errorf("mobility profile: M is %v, which is >= 0.0.\n\t%#v", mp.M, mp)
	}
	if mp.T != mp.B {
		t.Errorf("mobility profile: T of %v is NOT equal to B of %v, but they should be equal.\n\t%#v", mp.T, mp.B, mp)
	}
}
