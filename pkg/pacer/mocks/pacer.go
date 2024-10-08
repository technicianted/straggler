// Code generated by MockGen. DO NOT EDIT.
// Source: types.go
//
// Generated by this command:
//
//	mockgen -package mocks -destination ../mocks/pacer.go -source types.go
//

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"
	types "straggler/pkg/pacer/types"

	logr "github.com/go-logr/logr"
	gomock "go.uber.org/mock/gomock"
	v1 "k8s.io/api/core/v1"
)

// MockPacer is a mock of Pacer interface.
type MockPacer struct {
	ctrl     *gomock.Controller
	recorder *MockPacerMockRecorder
}

// MockPacerMockRecorder is the mock recorder for MockPacer.
type MockPacerMockRecorder struct {
	mock *MockPacer
}

// NewMockPacer creates a new mock instance.
func NewMockPacer(ctrl *gomock.Controller) *MockPacer {
	mock := &MockPacer{ctrl: ctrl}
	mock.recorder = &MockPacerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPacer) EXPECT() *MockPacerMockRecorder {
	return m.recorder
}

// ID mocks base method.
func (m *MockPacer) ID() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ID")
	ret0, _ := ret[0].(string)
	return ret0
}

// ID indicates an expected call of ID.
func (mr *MockPacerMockRecorder) ID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ID", reflect.TypeOf((*MockPacer)(nil).ID))
}

// Pace mocks base method.
func (m *MockPacer) Pace(podClassifications types.PodClassification, logger logr.Logger) ([]v1.Pod, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Pace", podClassifications, logger)
	ret0, _ := ret[0].([]v1.Pod)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Pace indicates an expected call of Pace.
func (mr *MockPacerMockRecorder) Pace(podClassifications, logger any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Pace", reflect.TypeOf((*MockPacer)(nil).Pace), podClassifications, logger)
}

// MockPacerFactory is a mock of PacerFactory interface.
type MockPacerFactory struct {
	ctrl     *gomock.Controller
	recorder *MockPacerFactoryMockRecorder
}

// MockPacerFactoryMockRecorder is the mock recorder for MockPacerFactory.
type MockPacerFactoryMockRecorder struct {
	mock *MockPacerFactory
}

// NewMockPacerFactory creates a new mock instance.
func NewMockPacerFactory(ctrl *gomock.Controller) *MockPacerFactory {
	mock := &MockPacerFactory{ctrl: ctrl}
	mock.recorder = &MockPacerFactoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPacerFactory) EXPECT() *MockPacerFactoryMockRecorder {
	return m.recorder
}

// New mocks base method.
func (m *MockPacerFactory) New(key string) types.Pacer {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "New", key)
	ret0, _ := ret[0].(types.Pacer)
	return ret0
}

// New indicates an expected call of New.
func (mr *MockPacerFactoryMockRecorder) New(key any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "New", reflect.TypeOf((*MockPacerFactory)(nil).New), key)
}
