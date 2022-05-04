package extcombiner

import "runtime"

func spin(v *int) {
	*v++
	if *v >= 8 {
		runtime.Gosched()
	}
}

const busyspin = 8
