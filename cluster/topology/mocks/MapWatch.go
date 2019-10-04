// Code generated by mockery v1.0.0
package mocks

import mock "github.com/stretchr/testify/mock"
import topology "github.com/uber/aresdb/cluster/topology"

// MapWatch is an autogenerated mock type for the MapWatch type
type MapWatch struct {
	mock.Mock
}

// C provides a mock function with given fields:
func (_m *MapWatch) C() <-chan struct{} {
	ret := _m.Called()

	var r0 <-chan struct{}
	if rf, ok := ret.Get(0).(func() <-chan struct{}); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(<-chan struct{})
		}
	}

	return r0
}

// Close provides a mock function with given fields:
func (_m *MapWatch) Close() {
	_m.Called()
}

// Get provides a mock function with given fields:
func (_m *MapWatch) Get() topology.Map {
	ret := _m.Called()

	var r0 topology.Map
	if rf, ok := ret.Get(0).(func() topology.Map); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(topology.Map)
		}
	}

	return r0
}
