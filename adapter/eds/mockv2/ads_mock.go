// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2 (interfaces: AggregatedDiscoveryServiceClient,AggregatedDiscoveryService_StreamAggregatedResourcesClient)

// Package mock_v2 is a generated GoMock package.
package mock_v2

import (
	context "context"
	reflect "reflect"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	v20 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	gomock "github.com/golang/mock/gomock"
	grpc "google.golang.org/grpc"
	metadata "google.golang.org/grpc/metadata"
)

// MockAggregatedDiscoveryServiceClient is a mock of AggregatedDiscoveryServiceClient interface
type MockAggregatedDiscoveryServiceClient struct {
	ctrl     *gomock.Controller
	recorder *MockAggregatedDiscoveryServiceClientMockRecorder
}

// MockAggregatedDiscoveryServiceClientMockRecorder is the mock recorder for MockAggregatedDiscoveryServiceClient
type MockAggregatedDiscoveryServiceClientMockRecorder struct {
	mock *MockAggregatedDiscoveryServiceClient
}

// NewMockAggregatedDiscoveryServiceClient creates a new mock instance
func NewMockAggregatedDiscoveryServiceClient(ctrl *gomock.Controller) *MockAggregatedDiscoveryServiceClient {
	mock := &MockAggregatedDiscoveryServiceClient{ctrl: ctrl}
	mock.recorder = &MockAggregatedDiscoveryServiceClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockAggregatedDiscoveryServiceClient) EXPECT() *MockAggregatedDiscoveryServiceClientMockRecorder {
	return m.recorder
}

// DeltaAggregatedResources mocks base method
func (m *MockAggregatedDiscoveryServiceClient) DeltaAggregatedResources(arg0 context.Context, arg1 ...grpc.CallOption) (v20.AggregatedDiscoveryService_DeltaAggregatedResourcesClient, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0}
	for _, a := range arg1 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "DeltaAggregatedResources", varargs...)
	ret0, _ := ret[0].(v20.AggregatedDiscoveryService_DeltaAggregatedResourcesClient)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DeltaAggregatedResources indicates an expected call of DeltaAggregatedResources
func (mr *MockAggregatedDiscoveryServiceClientMockRecorder) DeltaAggregatedResources(arg0 interface{}, arg1 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0}, arg1...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeltaAggregatedResources", reflect.TypeOf((*MockAggregatedDiscoveryServiceClient)(nil).DeltaAggregatedResources), varargs...)
}

// StreamAggregatedResources mocks base method
func (m *MockAggregatedDiscoveryServiceClient) StreamAggregatedResources(arg0 context.Context, arg1 ...grpc.CallOption) (v20.AggregatedDiscoveryService_StreamAggregatedResourcesClient, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0}
	for _, a := range arg1 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "StreamAggregatedResources", varargs...)
	ret0, _ := ret[0].(v20.AggregatedDiscoveryService_StreamAggregatedResourcesClient)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// StreamAggregatedResources indicates an expected call of StreamAggregatedResources
func (mr *MockAggregatedDiscoveryServiceClientMockRecorder) StreamAggregatedResources(arg0 interface{}, arg1 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0}, arg1...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StreamAggregatedResources", reflect.TypeOf((*MockAggregatedDiscoveryServiceClient)(nil).StreamAggregatedResources), varargs...)
}

// MockAggregatedDiscoveryService_StreamAggregatedResourcesClient is a mock of AggregatedDiscoveryService_StreamAggregatedResourcesClient interface
type MockAggregatedDiscoveryService_StreamAggregatedResourcesClient struct {
	ctrl     *gomock.Controller
	recorder *MockAggregatedDiscoveryService_StreamAggregatedResourcesClientMockRecorder
}

// MockAggregatedDiscoveryService_StreamAggregatedResourcesClientMockRecorder is the mock recorder for MockAggregatedDiscoveryService_StreamAggregatedResourcesClient
type MockAggregatedDiscoveryService_StreamAggregatedResourcesClientMockRecorder struct {
	mock *MockAggregatedDiscoveryService_StreamAggregatedResourcesClient
}

// NewMockAggregatedDiscoveryService_StreamAggregatedResourcesClient creates a new mock instance
func NewMockAggregatedDiscoveryService_StreamAggregatedResourcesClient(ctrl *gomock.Controller) *MockAggregatedDiscoveryService_StreamAggregatedResourcesClient {
	mock := &MockAggregatedDiscoveryService_StreamAggregatedResourcesClient{ctrl: ctrl}
	mock.recorder = &MockAggregatedDiscoveryService_StreamAggregatedResourcesClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockAggregatedDiscoveryService_StreamAggregatedResourcesClient) EXPECT() *MockAggregatedDiscoveryService_StreamAggregatedResourcesClientMockRecorder {
	return m.recorder
}

// CloseSend mocks base method
func (m *MockAggregatedDiscoveryService_StreamAggregatedResourcesClient) CloseSend() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CloseSend")
	ret0, _ := ret[0].(error)
	return ret0
}

// CloseSend indicates an expected call of CloseSend
func (mr *MockAggregatedDiscoveryService_StreamAggregatedResourcesClientMockRecorder) CloseSend() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CloseSend", reflect.TypeOf((*MockAggregatedDiscoveryService_StreamAggregatedResourcesClient)(nil).CloseSend))
}

// Context mocks base method
func (m *MockAggregatedDiscoveryService_StreamAggregatedResourcesClient) Context() context.Context {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Context")
	ret0, _ := ret[0].(context.Context)
	return ret0
}

// Context indicates an expected call of Context
func (mr *MockAggregatedDiscoveryService_StreamAggregatedResourcesClientMockRecorder) Context() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Context", reflect.TypeOf((*MockAggregatedDiscoveryService_StreamAggregatedResourcesClient)(nil).Context))
}

// Header mocks base method
func (m *MockAggregatedDiscoveryService_StreamAggregatedResourcesClient) Header() (metadata.MD, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Header")
	ret0, _ := ret[0].(metadata.MD)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Header indicates an expected call of Header
func (mr *MockAggregatedDiscoveryService_StreamAggregatedResourcesClientMockRecorder) Header() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Header", reflect.TypeOf((*MockAggregatedDiscoveryService_StreamAggregatedResourcesClient)(nil).Header))
}

// Recv mocks base method
func (m *MockAggregatedDiscoveryService_StreamAggregatedResourcesClient) Recv() (*v2.DiscoveryResponse, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Recv")
	ret0, _ := ret[0].(*v2.DiscoveryResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Recv indicates an expected call of Recv
func (mr *MockAggregatedDiscoveryService_StreamAggregatedResourcesClientMockRecorder) Recv() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Recv", reflect.TypeOf((*MockAggregatedDiscoveryService_StreamAggregatedResourcesClient)(nil).Recv))
}

// RecvMsg mocks base method
func (m *MockAggregatedDiscoveryService_StreamAggregatedResourcesClient) RecvMsg(arg0 interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RecvMsg", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// RecvMsg indicates an expected call of RecvMsg
func (mr *MockAggregatedDiscoveryService_StreamAggregatedResourcesClientMockRecorder) RecvMsg(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RecvMsg", reflect.TypeOf((*MockAggregatedDiscoveryService_StreamAggregatedResourcesClient)(nil).RecvMsg), arg0)
}

// Send mocks base method
func (m *MockAggregatedDiscoveryService_StreamAggregatedResourcesClient) Send(arg0 *v2.DiscoveryRequest) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Send", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Send indicates an expected call of Send
func (mr *MockAggregatedDiscoveryService_StreamAggregatedResourcesClientMockRecorder) Send(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Send", reflect.TypeOf((*MockAggregatedDiscoveryService_StreamAggregatedResourcesClient)(nil).Send), arg0)
}

// SendMsg mocks base method
func (m *MockAggregatedDiscoveryService_StreamAggregatedResourcesClient) SendMsg(arg0 interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendMsg", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendMsg indicates an expected call of SendMsg
func (mr *MockAggregatedDiscoveryService_StreamAggregatedResourcesClientMockRecorder) SendMsg(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMsg", reflect.TypeOf((*MockAggregatedDiscoveryService_StreamAggregatedResourcesClient)(nil).SendMsg), arg0)
}

// Trailer mocks base method
func (m *MockAggregatedDiscoveryService_StreamAggregatedResourcesClient) Trailer() metadata.MD {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Trailer")
	ret0, _ := ret[0].(metadata.MD)
	return ret0
}

// Trailer indicates an expected call of Trailer
func (mr *MockAggregatedDiscoveryService_StreamAggregatedResourcesClientMockRecorder) Trailer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Trailer", reflect.TypeOf((*MockAggregatedDiscoveryService_StreamAggregatedResourcesClient)(nil).Trailer))
}
