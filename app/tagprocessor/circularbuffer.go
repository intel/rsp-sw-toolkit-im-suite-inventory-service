package tagprocessor

type CircularBuffer struct {
	windowSize int
	values     []float64
	counter    int
}

func NewCircularBuffer(windowSize int) *CircularBuffer {
	return &CircularBuffer{
		windowSize: windowSize,
		values:     make([]float64, windowSize),
	}
}

func (buff *CircularBuffer) getN() int {
	if buff.counter >= buff.windowSize {
		return buff.windowSize
	}
	return buff.counter
}

func (buff *CircularBuffer) getMean() float64 {
	n := buff.getN()
	var total float64
	for i := 0; i < n; i++ {
		total += buff.values[i]
	}
	return total / float64(n)
}

func (buff *CircularBuffer) addValue(value float64) {
	buff.values[buff.counter%buff.windowSize] = value
	buff.counter++
}
