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

// GetCount returns the number of actual values present in the buffer
func (buff *CircularBuffer) GetCount() int {
	if buff.counter >= buff.windowSize {
		return buff.windowSize
	}
	return buff.counter
}

func (buff *CircularBuffer) GetMean() float64 {
	count := buff.GetCount()
	var total float64
	for i := 0; i < count; i++ {
		total += buff.values[i]
	}
	return total / float64(count)
}

func (buff *CircularBuffer) AddValue(value float64) {
	buff.values[buff.counter%buff.windowSize] = value
	buff.counter++
}
