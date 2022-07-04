// Code generated by mockery v2.12.3. DO NOT EDIT.

package mocks

import (
	compliance "github.com/DataDog/datadog-agent/pkg/compliance"
	mock "github.com/stretchr/testify/mock"
)

// Builder is an autogenerated mock type for the Builder type
type Builder struct {
	mock.Mock
}

// ChecksFromFile provides a mock function with given fields: file, onCheck
func (_m *Builder) ChecksFromFile(file string, onCheck compliance.CheckVisitor) error {
	ret := _m.Called(file, onCheck)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, compliance.CheckVisitor) error); ok {
		r0 = rf(file, onCheck)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Close provides a mock function with given fields:
func (_m *Builder) Close() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetCheckStatus provides a mock function with given fields:
func (_m *Builder) GetCheckStatus() compliance.CheckStatusList {
	ret := _m.Called()

	var r0 compliance.CheckStatusList
	if rf, ok := ret.Get(0).(func() compliance.CheckStatusList); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(compliance.CheckStatusList)
		}
	}

	return r0
}

type NewBuilderT interface {
	mock.TestingT
	Cleanup(func())
}

// NewBuilder creates a new instance of Builder. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewBuilder(t NewBuilderT) *Builder {
	mock := &Builder{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
