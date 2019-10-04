package jsonrpc

type SensorDeviceIdsResponse []string

// implement the jsonrpc.Message interface
func (info *SensorDeviceIdsResponse) Validate() error {
	return nil
}
