package flightv1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

const _ = grpc.SupportPackageIsVersion9

const (
	FlightService_SearchFlights_FullMethodName      = "/flight.v1.FlightService/SearchFlights"
	FlightService_GetFlight_FullMethodName          = "/flight.v1.FlightService/GetFlight"
	FlightService_ReserveSeats_FullMethodName       = "/flight.v1.FlightService/ReserveSeats"
	FlightService_ReleaseReservation_FullMethodName = "/flight.v1.FlightService/ReleaseReservation"
)

type FlightServiceClient interface {
	SearchFlights(ctx context.Context, in *SearchFlightsRequest, opts ...grpc.CallOption) (*SearchFlightsResponse, error)
	GetFlight(ctx context.Context, in *GetFlightRequest, opts ...grpc.CallOption) (*Flight, error)
	ReserveSeats(ctx context.Context, in *ReserveSeatsRequest, opts ...grpc.CallOption) (*SeatReservation, error)
	ReleaseReservation(ctx context.Context, in *ReleaseReservationRequest, opts ...grpc.CallOption) (*SeatReservation, error)
}

type flightServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewFlightServiceClient(cc grpc.ClientConnInterface) FlightServiceClient {
	return &flightServiceClient{cc}
}

func (c *flightServiceClient) SearchFlights(ctx context.Context, in *SearchFlightsRequest, opts ...grpc.CallOption) (*SearchFlightsResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(SearchFlightsResponse)
	err := c.cc.Invoke(ctx, FlightService_SearchFlights_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *flightServiceClient) GetFlight(ctx context.Context, in *GetFlightRequest, opts ...grpc.CallOption) (*Flight, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Flight)
	err := c.cc.Invoke(ctx, FlightService_GetFlight_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *flightServiceClient) ReserveSeats(ctx context.Context, in *ReserveSeatsRequest, opts ...grpc.CallOption) (*SeatReservation, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(SeatReservation)
	err := c.cc.Invoke(ctx, FlightService_ReserveSeats_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *flightServiceClient) ReleaseReservation(ctx context.Context, in *ReleaseReservationRequest, opts ...grpc.CallOption) (*SeatReservation, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(SeatReservation)
	err := c.cc.Invoke(ctx, FlightService_ReleaseReservation_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type FlightServiceServer interface {
	SearchFlights(context.Context, *SearchFlightsRequest) (*SearchFlightsResponse, error)
	GetFlight(context.Context, *GetFlightRequest) (*Flight, error)
	ReserveSeats(context.Context, *ReserveSeatsRequest) (*SeatReservation, error)
	ReleaseReservation(context.Context, *ReleaseReservationRequest) (*SeatReservation, error)
	mustEmbedUnimplementedFlightServiceServer()
}

type UnimplementedFlightServiceServer struct{}

func (UnimplementedFlightServiceServer) SearchFlights(context.Context, *SearchFlightsRequest) (*SearchFlightsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method SearchFlights not implemented")
}
func (UnimplementedFlightServiceServer) GetFlight(context.Context, *GetFlightRequest) (*Flight, error) {
	return nil, status.Error(codes.Unimplemented, "method GetFlight not implemented")
}
func (UnimplementedFlightServiceServer) ReserveSeats(context.Context, *ReserveSeatsRequest) (*SeatReservation, error) {
	return nil, status.Error(codes.Unimplemented, "method ReserveSeats not implemented")
}
func (UnimplementedFlightServiceServer) ReleaseReservation(context.Context, *ReleaseReservationRequest) (*SeatReservation, error) {
	return nil, status.Error(codes.Unimplemented, "method ReleaseReservation not implemented")
}
func (UnimplementedFlightServiceServer) mustEmbedUnimplementedFlightServiceServer() {}
func (UnimplementedFlightServiceServer) testEmbeddedByValue()                       {}

type UnsafeFlightServiceServer interface {
	mustEmbedUnimplementedFlightServiceServer()
}

func RegisterFlightServiceServer(s grpc.ServiceRegistrar, srv FlightServiceServer) {

	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&FlightService_ServiceDesc, srv)
}

func _FlightService_SearchFlights_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SearchFlightsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FlightServiceServer).SearchFlights(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: FlightService_SearchFlights_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FlightServiceServer).SearchFlights(ctx, req.(*SearchFlightsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FlightService_GetFlight_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetFlightRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FlightServiceServer).GetFlight(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: FlightService_GetFlight_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FlightServiceServer).GetFlight(ctx, req.(*GetFlightRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FlightService_ReserveSeats_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReserveSeatsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FlightServiceServer).ReserveSeats(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: FlightService_ReserveSeats_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FlightServiceServer).ReserveSeats(ctx, req.(*ReserveSeatsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _FlightService_ReleaseReservation_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReleaseReservationRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FlightServiceServer).ReleaseReservation(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: FlightService_ReleaseReservation_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FlightServiceServer).ReleaseReservation(ctx, req.(*ReleaseReservationRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var FlightService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "flight.v1.FlightService",
	HandlerType: (*FlightServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "SearchFlights",
			Handler:    _FlightService_SearchFlights_Handler,
		},
		{
			MethodName: "GetFlight",
			Handler:    _FlightService_GetFlight_Handler,
		},
		{
			MethodName: "ReserveSeats",
			Handler:    _FlightService_ReserveSeats_Handler,
		},
		{
			MethodName: "ReleaseReservation",
			Handler:    _FlightService_ReleaseReservation_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "api/flight/v1/flight.proto",
}
