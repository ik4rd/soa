package flightv1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)

	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type FlightStatus int32

const (
	FlightStatus_FLIGHT_STATUS_UNSPECIFIED FlightStatus = 0
	FlightStatus_FLIGHT_STATUS_SCHEDULED   FlightStatus = 1
	FlightStatus_FLIGHT_STATUS_DEPARTED    FlightStatus = 2
	FlightStatus_FLIGHT_STATUS_CANCELLED   FlightStatus = 3
	FlightStatus_FLIGHT_STATUS_COMPLETED   FlightStatus = 4
)

var (
	FlightStatus_name = map[int32]string{
		0: "FLIGHT_STATUS_UNSPECIFIED",
		1: "FLIGHT_STATUS_SCHEDULED",
		2: "FLIGHT_STATUS_DEPARTED",
		3: "FLIGHT_STATUS_CANCELLED",
		4: "FLIGHT_STATUS_COMPLETED",
	}
	FlightStatus_value = map[string]int32{
		"FLIGHT_STATUS_UNSPECIFIED": 0,
		"FLIGHT_STATUS_SCHEDULED":   1,
		"FLIGHT_STATUS_DEPARTED":    2,
		"FLIGHT_STATUS_CANCELLED":   3,
		"FLIGHT_STATUS_COMPLETED":   4,
	}
)

func (x FlightStatus) Enum() *FlightStatus {
	p := new(FlightStatus)
	*p = x
	return p
}

func (x FlightStatus) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (FlightStatus) Descriptor() protoreflect.EnumDescriptor {
	return file_api_flight_v1_flight_proto_enumTypes[0].Descriptor()
}

func (FlightStatus) Type() protoreflect.EnumType {
	return &file_api_flight_v1_flight_proto_enumTypes[0]
}

func (x FlightStatus) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

func (FlightStatus) EnumDescriptor() ([]byte, []int) {
	return file_api_flight_v1_flight_proto_rawDescGZIP(), []int{0}
}

type ReservationStatus int32

const (
	ReservationStatus_RESERVATION_STATUS_UNSPECIFIED ReservationStatus = 0
	ReservationStatus_RESERVATION_STATUS_ACTIVE      ReservationStatus = 1
	ReservationStatus_RESERVATION_STATUS_RELEASED    ReservationStatus = 2
	ReservationStatus_RESERVATION_STATUS_EXPIRED     ReservationStatus = 3
)

var (
	ReservationStatus_name = map[int32]string{
		0: "RESERVATION_STATUS_UNSPECIFIED",
		1: "RESERVATION_STATUS_ACTIVE",
		2: "RESERVATION_STATUS_RELEASED",
		3: "RESERVATION_STATUS_EXPIRED",
	}
	ReservationStatus_value = map[string]int32{
		"RESERVATION_STATUS_UNSPECIFIED": 0,
		"RESERVATION_STATUS_ACTIVE":      1,
		"RESERVATION_STATUS_RELEASED":    2,
		"RESERVATION_STATUS_EXPIRED":     3,
	}
)

func (x ReservationStatus) Enum() *ReservationStatus {
	p := new(ReservationStatus)
	*p = x
	return p
}

func (x ReservationStatus) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (ReservationStatus) Descriptor() protoreflect.EnumDescriptor {
	return file_api_flight_v1_flight_proto_enumTypes[1].Descriptor()
}

func (ReservationStatus) Type() protoreflect.EnumType {
	return &file_api_flight_v1_flight_proto_enumTypes[1]
}

func (x ReservationStatus) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

func (ReservationStatus) EnumDescriptor() ([]byte, []int) {
	return file_api_flight_v1_flight_proto_rawDescGZIP(), []int{1}
}

type Flight struct {
	state          protoimpl.MessageState `protogen:"open.v1"`
	Id             string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	FlightNumber   string                 `protobuf:"bytes,2,opt,name=flight_number,json=flightNumber,proto3" json:"flight_number,omitempty"`
	AirlineCode    string                 `protobuf:"bytes,3,opt,name=airline_code,json=airlineCode,proto3" json:"airline_code,omitempty"`
	Origin         string                 `protobuf:"bytes,4,opt,name=origin,proto3" json:"origin,omitempty"`
	Destination    string                 `protobuf:"bytes,5,opt,name=destination,proto3" json:"destination,omitempty"`
	DepartureAt    *timestamppb.Timestamp `protobuf:"bytes,6,opt,name=departure_at,json=departureAt,proto3" json:"departure_at,omitempty"`
	ArrivalAt      *timestamppb.Timestamp `protobuf:"bytes,7,opt,name=arrival_at,json=arrivalAt,proto3" json:"arrival_at,omitempty"`
	TotalSeats     int32                  `protobuf:"varint,8,opt,name=total_seats,json=totalSeats,proto3" json:"total_seats,omitempty"`
	AvailableSeats int32                  `protobuf:"varint,9,opt,name=available_seats,json=availableSeats,proto3" json:"available_seats,omitempty"`
	PriceMinor     int64                  `protobuf:"varint,10,opt,name=price_minor,json=priceMinor,proto3" json:"price_minor,omitempty"`
	Status         FlightStatus           `protobuf:"varint,11,opt,name=status,proto3,enum=flight.v1.FlightStatus" json:"status,omitempty"`
	unknownFields  protoimpl.UnknownFields
	sizeCache      protoimpl.SizeCache
}

func (x *Flight) Reset() {
	*x = Flight{}
	mi := &file_api_flight_v1_flight_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Flight) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Flight) ProtoMessage() {}

func (x *Flight) ProtoReflect() protoreflect.Message {
	mi := &file_api_flight_v1_flight_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*Flight) Descriptor() ([]byte, []int) {
	return file_api_flight_v1_flight_proto_rawDescGZIP(), []int{0}
}

func (x *Flight) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *Flight) GetFlightNumber() string {
	if x != nil {
		return x.FlightNumber
	}
	return ""
}

func (x *Flight) GetAirlineCode() string {
	if x != nil {
		return x.AirlineCode
	}
	return ""
}

func (x *Flight) GetOrigin() string {
	if x != nil {
		return x.Origin
	}
	return ""
}

func (x *Flight) GetDestination() string {
	if x != nil {
		return x.Destination
	}
	return ""
}

func (x *Flight) GetDepartureAt() *timestamppb.Timestamp {
	if x != nil {
		return x.DepartureAt
	}
	return nil
}

func (x *Flight) GetArrivalAt() *timestamppb.Timestamp {
	if x != nil {
		return x.ArrivalAt
	}
	return nil
}

func (x *Flight) GetTotalSeats() int32 {
	if x != nil {
		return x.TotalSeats
	}
	return 0
}

func (x *Flight) GetAvailableSeats() int32 {
	if x != nil {
		return x.AvailableSeats
	}
	return 0
}

func (x *Flight) GetPriceMinor() int64 {
	if x != nil {
		return x.PriceMinor
	}
	return 0
}

func (x *Flight) GetStatus() FlightStatus {
	if x != nil {
		return x.Status
	}
	return FlightStatus_FLIGHT_STATUS_UNSPECIFIED
}

type SeatReservation struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	BookingId     string                 `protobuf:"bytes,1,opt,name=booking_id,json=bookingId,proto3" json:"booking_id,omitempty"`
	FlightId      string                 `protobuf:"bytes,2,opt,name=flight_id,json=flightId,proto3" json:"flight_id,omitempty"`
	SeatCount     int32                  `protobuf:"varint,3,opt,name=seat_count,json=seatCount,proto3" json:"seat_count,omitempty"`
	Status        ReservationStatus      `protobuf:"varint,4,opt,name=status,proto3,enum=flight.v1.ReservationStatus" json:"status,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *SeatReservation) Reset() {
	*x = SeatReservation{}
	mi := &file_api_flight_v1_flight_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SeatReservation) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SeatReservation) ProtoMessage() {}

func (x *SeatReservation) ProtoReflect() protoreflect.Message {
	mi := &file_api_flight_v1_flight_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*SeatReservation) Descriptor() ([]byte, []int) {
	return file_api_flight_v1_flight_proto_rawDescGZIP(), []int{1}
}

func (x *SeatReservation) GetBookingId() string {
	if x != nil {
		return x.BookingId
	}
	return ""
}

func (x *SeatReservation) GetFlightId() string {
	if x != nil {
		return x.FlightId
	}
	return ""
}

func (x *SeatReservation) GetSeatCount() int32 {
	if x != nil {
		return x.SeatCount
	}
	return 0
}

func (x *SeatReservation) GetStatus() ReservationStatus {
	if x != nil {
		return x.Status
	}
	return ReservationStatus_RESERVATION_STATUS_UNSPECIFIED
}

type SearchFlightsRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Origin        string                 `protobuf:"bytes,1,opt,name=origin,proto3" json:"origin,omitempty"`
	Destination   string                 `protobuf:"bytes,2,opt,name=destination,proto3" json:"destination,omitempty"`
	Date          *timestamppb.Timestamp `protobuf:"bytes,3,opt,name=date,proto3" json:"date,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *SearchFlightsRequest) Reset() {
	*x = SearchFlightsRequest{}
	mi := &file_api_flight_v1_flight_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SearchFlightsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SearchFlightsRequest) ProtoMessage() {}

func (x *SearchFlightsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_api_flight_v1_flight_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*SearchFlightsRequest) Descriptor() ([]byte, []int) {
	return file_api_flight_v1_flight_proto_rawDescGZIP(), []int{2}
}

func (x *SearchFlightsRequest) GetOrigin() string {
	if x != nil {
		return x.Origin
	}
	return ""
}

func (x *SearchFlightsRequest) GetDestination() string {
	if x != nil {
		return x.Destination
	}
	return ""
}

func (x *SearchFlightsRequest) GetDate() *timestamppb.Timestamp {
	if x != nil {
		return x.Date
	}
	return nil
}

type SearchFlightsResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Flights       []*Flight              `protobuf:"bytes,1,rep,name=flights,proto3" json:"flights,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *SearchFlightsResponse) Reset() {
	*x = SearchFlightsResponse{}
	mi := &file_api_flight_v1_flight_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *SearchFlightsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SearchFlightsResponse) ProtoMessage() {}

func (x *SearchFlightsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_api_flight_v1_flight_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*SearchFlightsResponse) Descriptor() ([]byte, []int) {
	return file_api_flight_v1_flight_proto_rawDescGZIP(), []int{3}
}

func (x *SearchFlightsResponse) GetFlights() []*Flight {
	if x != nil {
		return x.Flights
	}
	return nil
}

type GetFlightRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	FlightId      string                 `protobuf:"bytes,1,opt,name=flight_id,json=flightId,proto3" json:"flight_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetFlightRequest) Reset() {
	*x = GetFlightRequest{}
	mi := &file_api_flight_v1_flight_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetFlightRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetFlightRequest) ProtoMessage() {}

func (x *GetFlightRequest) ProtoReflect() protoreflect.Message {
	mi := &file_api_flight_v1_flight_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*GetFlightRequest) Descriptor() ([]byte, []int) {
	return file_api_flight_v1_flight_proto_rawDescGZIP(), []int{4}
}

func (x *GetFlightRequest) GetFlightId() string {
	if x != nil {
		return x.FlightId
	}
	return ""
}

type ReserveSeatsRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	FlightId      string                 `protobuf:"bytes,1,opt,name=flight_id,json=flightId,proto3" json:"flight_id,omitempty"`
	SeatCount     int32                  `protobuf:"varint,2,opt,name=seat_count,json=seatCount,proto3" json:"seat_count,omitempty"`
	BookingId     string                 `protobuf:"bytes,3,opt,name=booking_id,json=bookingId,proto3" json:"booking_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ReserveSeatsRequest) Reset() {
	*x = ReserveSeatsRequest{}
	mi := &file_api_flight_v1_flight_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ReserveSeatsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReserveSeatsRequest) ProtoMessage() {}

func (x *ReserveSeatsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_api_flight_v1_flight_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*ReserveSeatsRequest) Descriptor() ([]byte, []int) {
	return file_api_flight_v1_flight_proto_rawDescGZIP(), []int{5}
}

func (x *ReserveSeatsRequest) GetFlightId() string {
	if x != nil {
		return x.FlightId
	}
	return ""
}

func (x *ReserveSeatsRequest) GetSeatCount() int32 {
	if x != nil {
		return x.SeatCount
	}
	return 0
}

func (x *ReserveSeatsRequest) GetBookingId() string {
	if x != nil {
		return x.BookingId
	}
	return ""
}

type ReleaseReservationRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	BookingId     string                 `protobuf:"bytes,1,opt,name=booking_id,json=bookingId,proto3" json:"booking_id,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *ReleaseReservationRequest) Reset() {
	*x = ReleaseReservationRequest{}
	mi := &file_api_flight_v1_flight_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *ReleaseReservationRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReleaseReservationRequest) ProtoMessage() {}

func (x *ReleaseReservationRequest) ProtoReflect() protoreflect.Message {
	mi := &file_api_flight_v1_flight_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

func (*ReleaseReservationRequest) Descriptor() ([]byte, []int) {
	return file_api_flight_v1_flight_proto_rawDescGZIP(), []int{6}
}

func (x *ReleaseReservationRequest) GetBookingId() string {
	if x != nil {
		return x.BookingId
	}
	return ""
}

var File_api_flight_v1_flight_proto protoreflect.FileDescriptor

const file_api_flight_v1_flight_proto_rawDesc = "" +
	"\n" +
	"\x1aapi/flight/v1/flight.proto\x12\tflight.v1\x1a\x1fgoogle/protobuf/timestamp.proto\"\xb0\x03\n" +
	"\x06Flight\x12\x0e\n" +
	"\x02id\x18\x01 \x01(\tR\x02id\x12#\n" +
	"\rflight_number\x18\x02 \x01(\tR\fflightNumber\x12!\n" +
	"\fairline_code\x18\x03 \x01(\tR\vairlineCode\x12\x16\n" +
	"\x06origin\x18\x04 \x01(\tR\x06origin\x12 \n" +
	"\vdestination\x18\x05 \x01(\tR\vdestination\x12=\n" +
	"\fdeparture_at\x18\x06 \x01(\v2\x1a.google.protobuf.TimestampR\vdepartureAt\x129\n" +
	"\n" +
	"arrival_at\x18\a \x01(\v2\x1a.google.protobuf.TimestampR\tarrivalAt\x12\x1f\n" +
	"\vtotal_seats\x18\b \x01(\x05R\n" +
	"totalSeats\x12'\n" +
	"\x0favailable_seats\x18\t \x01(\x05R\x0eavailableSeats\x12\x1f\n" +
	"\vprice_minor\x18\n" +
	" \x01(\x03R\n" +
	"priceMinor\x12/\n" +
	"\x06status\x18\v \x01(\x0e2\x17.flight.v1.FlightStatusR\x06status\"\xa2\x01\n" +
	"\x0fSeatReservation\x12\x1d\n" +
	"\n" +
	"booking_id\x18\x01 \x01(\tR\tbookingId\x12\x1b\n" +
	"\tflight_id\x18\x02 \x01(\tR\bflightId\x12\x1d\n" +
	"\n" +
	"seat_count\x18\x03 \x01(\x05R\tseatCount\x124\n" +
	"\x06status\x18\x04 \x01(\x0e2\x1c.flight.v1.ReservationStatusR\x06status\"\x80\x01\n" +
	"\x14SearchFlightsRequest\x12\x16\n" +
	"\x06origin\x18\x01 \x01(\tR\x06origin\x12 \n" +
	"\vdestination\x18\x02 \x01(\tR\vdestination\x12.\n" +
	"\x04date\x18\x03 \x01(\v2\x1a.google.protobuf.TimestampR\x04date\"D\n" +
	"\x15SearchFlightsResponse\x12+\n" +
	"\aflights\x18\x01 \x03(\v2\x11.flight.v1.FlightR\aflights\"/\n" +
	"\x10GetFlightRequest\x12\x1b\n" +
	"\tflight_id\x18\x01 \x01(\tR\bflightId\"p\n" +
	"\x13ReserveSeatsRequest\x12\x1b\n" +
	"\tflight_id\x18\x01 \x01(\tR\bflightId\x12\x1d\n" +
	"\n" +
	"seat_count\x18\x02 \x01(\x05R\tseatCount\x12\x1d\n" +
	"\n" +
	"booking_id\x18\x03 \x01(\tR\tbookingId\":\n" +
	"\x19ReleaseReservationRequest\x12\x1d\n" +
	"\n" +
	"booking_id\x18\x01 \x01(\tR\tbookingId*\xa0\x01\n" +
	"\fFlightStatus\x12\x1d\n" +
	"\x19FLIGHT_STATUS_UNSPECIFIED\x10\x00\x12\x1b\n" +
	"\x17FLIGHT_STATUS_SCHEDULED\x10\x01\x12\x1a\n" +
	"\x16FLIGHT_STATUS_DEPARTED\x10\x02\x12\x1b\n" +
	"\x17FLIGHT_STATUS_CANCELLED\x10\x03\x12\x1b\n" +
	"\x17FLIGHT_STATUS_COMPLETED\x10\x04*\x97\x01\n" +
	"\x11ReservationStatus\x12\"\n" +
	"\x1eRESERVATION_STATUS_UNSPECIFIED\x10\x00\x12\x1d\n" +
	"\x19RESERVATION_STATUS_ACTIVE\x10\x01\x12\x1f\n" +
	"\x1bRESERVATION_STATUS_RELEASED\x10\x02\x12\x1e\n" +
	"\x1aRESERVATION_STATUS_EXPIRED\x10\x032\xc4\x02\n" +
	"\rFlightService\x12R\n" +
	"\rSearchFlights\x12\x1f.flight.v1.SearchFlightsRequest\x1a .flight.v1.SearchFlightsResponse\x12;\n" +
	"\tGetFlight\x12\x1b.flight.v1.GetFlightRequest\x1a\x11.flight.v1.Flight\x12J\n" +
	"\fReserveSeats\x12\x1e.flight.v1.ReserveSeatsRequest\x1a\x1a.flight.v1.SeatReservation\x12V\n" +
	"\x12ReleaseReservation\x12$.flight.v1.ReleaseReservationRequest\x1a\x1a.flight.v1.SeatReservationB&Z$flightbooking/api/flight/v1;flightv1b\x06proto3"

var (
	file_api_flight_v1_flight_proto_rawDescOnce sync.Once
	file_api_flight_v1_flight_proto_rawDescData []byte
)

func file_api_flight_v1_flight_proto_rawDescGZIP() []byte {
	file_api_flight_v1_flight_proto_rawDescOnce.Do(func() {
		file_api_flight_v1_flight_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_api_flight_v1_flight_proto_rawDesc), len(file_api_flight_v1_flight_proto_rawDesc)))
	})
	return file_api_flight_v1_flight_proto_rawDescData
}

var file_api_flight_v1_flight_proto_enumTypes = make([]protoimpl.EnumInfo, 2)
var file_api_flight_v1_flight_proto_msgTypes = make([]protoimpl.MessageInfo, 7)
var file_api_flight_v1_flight_proto_goTypes = []any{
	(FlightStatus)(0),
	(ReservationStatus)(0),
	(*Flight)(nil),
	(*SeatReservation)(nil),
	(*SearchFlightsRequest)(nil),
	(*SearchFlightsResponse)(nil),
	(*GetFlightRequest)(nil),
	(*ReserveSeatsRequest)(nil),
	(*ReleaseReservationRequest)(nil),
	(*timestamppb.Timestamp)(nil),
}
var file_api_flight_v1_flight_proto_depIdxs = []int32{
	9,
	9,
	0,
	1,
	9,
	2,
	4,
	6,
	7,
	8,
	5,
	2,
	3,
	3,
	10,
	6,
	6,
	6,
	0,
}

func init() { file_api_flight_v1_flight_proto_init() }
func file_api_flight_v1_flight_proto_init() {
	if File_api_flight_v1_flight_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_api_flight_v1_flight_proto_rawDesc), len(file_api_flight_v1_flight_proto_rawDesc)),
			NumEnums:      2,
			NumMessages:   7,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_api_flight_v1_flight_proto_goTypes,
		DependencyIndexes: file_api_flight_v1_flight_proto_depIdxs,
		EnumInfos:         file_api_flight_v1_flight_proto_enumTypes,
		MessageInfos:      file_api_flight_v1_flight_proto_msgTypes,
	}.Build()
	File_api_flight_v1_flight_proto = out.File
	file_api_flight_v1_flight_proto_goTypes = nil
	file_api_flight_v1_flight_proto_depIdxs = nil
}
