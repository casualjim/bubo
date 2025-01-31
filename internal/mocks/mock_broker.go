// Code generated by mockery v2.50.1. DO NOT EDIT.

package mocks

import (
	context "context"

	broker "github.com/casualjim/bubo/internal/broker"

	mock "github.com/stretchr/testify/mock"
)

// Broker is an autogenerated mock type for the Broker type
type Broker struct {
	mock.Mock
}

type Broker_Expecter struct {
	mock *mock.Mock
}

func (_m *Broker) EXPECT() *Broker_Expecter {
	return &Broker_Expecter{mock: &_m.Mock}
}

// Topic provides a mock function with given fields: _a0, _a1
func (_m *Broker) Topic(_a0 context.Context, _a1 string) broker.Topic {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for Topic")
	}

	var r0 broker.Topic
	if rf, ok := ret.Get(0).(func(context.Context, string) broker.Topic); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(broker.Topic)
		}
	}

	return r0
}

// Broker_Topic_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Topic'
type Broker_Topic_Call struct {
	*mock.Call
}

// Topic is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 string
func (_e *Broker_Expecter) Topic(_a0 interface{}, _a1 interface{}) *Broker_Topic_Call {
	return &Broker_Topic_Call{Call: _e.mock.On("Topic", _a0, _a1)}
}

func (_c *Broker_Topic_Call) Run(run func(_a0 context.Context, _a1 string)) *Broker_Topic_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *Broker_Topic_Call) Return(_a0 broker.Topic) *Broker_Topic_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *Broker_Topic_Call) RunAndReturn(run func(context.Context, string) broker.Topic) *Broker_Topic_Call {
	_c.Call.Return(run)
	return _c
}

// NewBroker creates a new instance of Broker. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewBroker(t interface {
	mock.TestingT
	Cleanup(func())
},
) *Broker {
	mock := &Broker{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
