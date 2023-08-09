// Package cartofacade contains the api to call into CGO
//
//nolint:lll
package cartofacade

/*
	#cgo CFLAGS: -I../viam-cartographer/src/carto_facade

	// the libraries that need to be linked can be derived from line 258 of the build.ninja file that is autogenerated during make build
	#cgo LDFLAGS: -L../viam-cartographer/build -L../viam-cartographer/build/cartographer -lviam-cartographer  -lcartographer -ldl -lm -labsl_hash  -labsl_city -labsl_bad_optional_access -labsl_strerror  -labsl_str_format_internal -labsl_synchronization -labsl_strings -labsl_throw_delegate -lcairo -llua5.3 -lstdc++ -lceres -lprotobuf -lglog -lboost_filesystem -lboost_iostreams -lpcl_io -lpcl_common -labsl_raw_hash_set

	#include "../viam-cartographer/src/carto_facade/carto_facade.h"
*/
import "C"

import (
	"errors"
	"time"
	"unsafe"
)

// CartoLib holds the c type viam_carto_lib
type CartoLib struct {
	value *C.viam_carto_lib
}

// CartoLibInterface describes the method signatures that CartoLib must implement
type CartoLibInterface interface {
	Terminate() error
}

// SlamMode represents the lidar configuration
type SlamMode int64

const (
	// UnknownMode denotes an unknown slam mode
	UnknownMode SlamMode = iota
	// MappingMode denotes the slam algo is in mapping mode
	MappingMode
	// LocalizingMode denotes the slam algo is in localizing only mode
	LocalizingMode
	// UpdatingMode denotes the slam algo is in updating mode
	UpdatingMode
)

// Carto holds the c type viam_carto
type Carto struct {
	value *C.viam_carto
	SlamMode
}

// CartoInterface describes the method signatures that Carto must implement
type CartoInterface interface {
	start() error
	stop() error
	terminate() error
	addLidarReading(string, []byte, time.Time) error
	addIMUReading(string, imuReading, time.Time) error
	getPosition() (GetPosition, error)
	getPointCloudMap() ([]byte, error)
	getInternalState() ([]byte, error)
}

// GetPosition holds values returned from c to be processed later
type GetPosition struct {
	X float64
	Y float64
	Z float64

	Real float64
	Imag float64
	Jmag float64
	Kmag float64

	ComponentReference string
}

// imuReading holds values for linear acceleration and angular velocity to be converted into c
type imuReading struct {
	LinAccX float64
	LinAccY float64
	LinAccZ float64
	AngVelX float64
	AngVelY float64
	AngVelZ float64
}

// LidarConfig represents the lidar configuration
type LidarConfig int64

const (
	// TwoD LidarConfig denotes a 2d lidar
	TwoD LidarConfig = iota
	// ThreeD LidarConfig denotes a 3d lidar
	ThreeD
)

// CartoConfig contains config values from app
type CartoConfig struct {
	Camera             string
	MovementSensor     string
	MapRateSecond      int
	DataDir            string
	ComponentReference string
	LidarConfig        LidarConfig

	CloudStoryEnabled bool
	EnableMapping     bool
	ExistingMap       string
}

// CartoAlgoConfig contains config values from app
type CartoAlgoConfig struct {
	OptimizeOnStart      bool
	OptimizeEveryNNodes  int
	NumRangeData         int
	MissingDataRayLength float32
	MaxRange             float32
	MinRange             float32
	UseIMUData           bool
	MaxSubmapsToKeep     int
	FreshSubmapsCount    int
	MinCoveredArea       float64
	MinAddedSubmapsCount int
	OccupiedSpaceWeight  float64
	TranslationWeight    float64
	RotationWeight       float64
}

// NewLib calls viam_carto_lib_init and returns a pointer to a viam carto lib object.
func NewLib(miniloglevel, verbose int) (CartoLib, error) {
	var pVcl *C.viam_carto_lib
	status := C.viam_carto_lib_init(&pVcl, C.int(miniloglevel), C.int(verbose))
	if err := toError(status); err != nil {
		return CartoLib{}, err
	}

	vcl := CartoLib{value: pVcl}

	return vcl, nil
}

// Terminate calls viam_carto_lib_terminate to clean up memory for viam carto lib.
func (vcl *CartoLib) Terminate() error {
	status := C.viam_carto_lib_terminate(&vcl.value)
	if err := toError(status); err != nil {
		return err
	}
	return nil
}

func toSlamMode(cSlamMode C.int) SlamMode {
	switch cSlamMode {
	case C.VIAM_CARTO_SLAM_MODE_MAPPING:
		return MappingMode
	case C.VIAM_CARTO_SLAM_MODE_LOCALIZING:
		return LocalizingMode
	case C.VIAM_CARTO_SLAM_MODE_UPDATING:
		return UpdatingMode
	default:
		return UnknownMode
	}
}

// NewCarto calls viam_carto_init and returns a pointer to a viam carto object. vcl is only an interface to facilitate testing & that the only type vcl it is actually expected to have is a CartoLib
func NewCarto(cfg CartoConfig, acfg CartoAlgoConfig, vcl CartoLibInterface) (Carto, error) {
	var pVc *C.viam_carto

	vcc, err := getConfig(cfg)
	if err != nil {
		return Carto{}, err
	}
	vcac := toAlgoConfig(acfg)
	cl, ok := vcl.(*CartoLib)
	if !ok {
		return Carto{}, errors.New("cannot cast provided library to a CartoLib")
	}
	status := C.viam_carto_init(&pVc, cl.value, vcc, vcac)

	if err := toError(status); err != nil {
		return Carto{}, err
	}

	carto := Carto{value: pVc, SlamMode: toSlamMode(pVc.slam_mode)}

	return carto, nil
}

// Start is a wrapper for viam_carto_start
func (vc *Carto) start() error {
	status := C.viam_carto_start(vc.value)

	if err := toError(status); err != nil {
		return err
	}

	return nil
}

// Stop is a wrapper for viam_carto_stop
func (vc *Carto) stop() error {
	status := C.viam_carto_stop(vc.value)

	if err := toError(status); err != nil {
		return err
	}

	return nil
}

// Terminate calls viam_carto_terminate to clean up memory for viam carto
func (vc *Carto) terminate() error {
	status := C.viam_carto_terminate(&vc.value)

	if err := toError(status); err != nil {
		return err
	}

	return nil
}

// AddLidarReading is a wrapper for viam_carto_add_lidar_reading
func (vc *Carto) addLidarReading(lidar string, readings []byte, timestamp time.Time) error {
	value := toLidarReading(lidar, readings, timestamp)

	status := C.viam_carto_add_lidar_reading(vc.value, &value)

	if err := toError(status); err != nil {
		return err
	}

	status = C.viam_carto_add_lidar_reading_destroy(&value)
	if err := toError(status); err != nil {
		return err
	}

	return nil
}

// AddIMUReading is a wrapper for viam_carto_add_imu_reading
func (vc *Carto) addIMUReading(imu string, readings imuReading, timestamp time.Time) error {
	value := toIMUReading(imu, readings, timestamp)

	status := C.viam_carto_add_imu_reading(vc.value, &value)

	if err := toError(status); err != nil {
		return err
	}

	status = C.viam_carto_add_imu_reading_destroy(&value)
	if err := toError(status); err != nil {
		return err
	}

	return nil
}

// GetPosition is a wrapper for viam_carto_get_position
func (vc *Carto) getPosition() (GetPosition, error) {
	value := C.viam_carto_get_position_response{}

	status := C.viam_carto_get_position(vc.value, &value)

	if err := toError(status); err != nil {
		return GetPosition{}, err
	}

	getPosition := toGetPositionResponse(value)

	status = C.viam_carto_get_position_response_destroy(&value)
	if err := toError(status); err != nil {
		return GetPosition{}, err
	}

	return getPosition, nil
}

// GetPointCloudMap is a wrapper for viam_carto_get_point_cloud_map
func (vc *Carto) getPointCloudMap() ([]byte, error) {
	value := C.viam_carto_get_point_cloud_map_response{}

	status := C.viam_carto_get_point_cloud_map(vc.value, &value)

	if err := toError(status); err != nil {
		return nil, err
	}

	pcd := bstringToByteSlice(value.point_cloud_pcd)

	status = C.viam_carto_get_point_cloud_map_response_destroy(&value)
	if err := toError(status); err != nil {
		return nil, err
	}

	return pcd, nil
}

// GetInternalState is a wrapper for viam_carto_get_internal_state
func (vc *Carto) getInternalState() ([]byte, error) {
	value := C.viam_carto_get_internal_state_response{}

	status := C.viam_carto_get_internal_state(vc.value, &value)

	if err := toError(status); err != nil {
		return nil, err
	}

	interalState := bstringToByteSlice(value.internal_state)

	status = C.viam_carto_get_internal_state_response_destroy(&value)
	if err := toError(status); err != nil {
		return nil, err
	}

	return interalState, nil
}

// this function is only used for testing purposes, but needs to be in this file as CGo is not supported in go test files
func getTestGetPositionResponse() C.viam_carto_get_position_response {
	gpr := C.viam_carto_get_position_response{}

	gpr.x = C.double(100)
	gpr.y = C.double(200)
	gpr.z = C.double(300)

	gpr.imag = C.double(700)
	gpr.jmag = C.double(800)
	gpr.kmag = C.double(900)

	gpr.real = C.double(1100)

	gpr.component_reference = goStringToBstring("C++ component reference")

	return gpr
}

func bstringToGoString(bstr C.bstring) string {
	return C.GoStringN(C.bstr2cstr(bstr, 0), bstr.slen)
}

func goStringToBstring(goStr string) C.bstring {
	cstr := C.CString(goStr)
	defer C.free(unsafe.Pointer(cstr))
	return C.blk2bstr(unsafe.Pointer(cstr), C.int(len(goStr)))
}

func toLidarConfig(lidarConfig LidarConfig) (C.viam_carto_LIDAR_CONFIG, error) {
	switch lidarConfig {
	case TwoD:
		return C.VIAM_CARTO_TWO_D, nil
	case ThreeD:
		return C.VIAM_CARTO_THREE_D, nil
	default:
		return 0, errors.New("invalid lidar config value")
	}
}

func getConfig(cfg CartoConfig) (C.viam_carto_config, error) {
	vcc := C.viam_carto_config{}
	vcc.camera = goStringToBstring(cfg.Camera)
	vcc.movement_sensor = goStringToBstring(cfg.MovementSensor)

	lidarCfg, err := toLidarConfig(cfg.LidarConfig)
	if err != nil {
		return C.viam_carto_config{}, err
	}

	vcc.map_rate_sec = C.int(cfg.MapRateSecond)
	vcc.data_dir = goStringToBstring(cfg.DataDir)
	vcc.lidar_config = lidarCfg

	vcc.cloud_story_enabled = C.bool(cfg.CloudStoryEnabled)
	vcc.enable_mapping = C.bool(cfg.EnableMapping)
	vcc.existing_map = goStringToBstring(cfg.ExistingMap)

	return vcc, nil
}

func toAlgoConfig(acfg CartoAlgoConfig) C.viam_carto_algo_config {
	vcac := C.viam_carto_algo_config{}
	vcac.optimize_on_start = C.bool(acfg.OptimizeOnStart)
	vcac.optimize_every_n_nodes = C.int(acfg.OptimizeEveryNNodes)
	vcac.num_range_data = C.int(acfg.NumRangeData)
	vcac.missing_data_ray_length = C.float(acfg.MissingDataRayLength)
	vcac.max_range = C.float(acfg.MaxRange)
	vcac.min_range = C.float(acfg.MinRange)
	vcac.use_imu_data = C.bool(acfg.UseIMUData)
	vcac.max_submaps_to_keep = C.int(acfg.MaxSubmapsToKeep)
	vcac.fresh_submaps_count = C.int(acfg.FreshSubmapsCount)
	vcac.min_covered_area = C.double(acfg.MinCoveredArea)
	vcac.min_added_submaps_count = C.int(acfg.MinAddedSubmapsCount)
	vcac.occupied_space_weight = C.double(acfg.OccupiedSpaceWeight)
	vcac.translation_weight = C.double(acfg.TranslationWeight)
	vcac.rotation_weight = C.double(acfg.RotationWeight)
	return vcac
}

func toGetPositionResponse(value C.viam_carto_get_position_response) GetPosition {
	return GetPosition{
		X: float64(value.x),
		Y: float64(value.y),
		Z: float64(value.z),

		Real: float64(value.real),
		Imag: float64(value.imag),
		Jmag: float64(value.jmag),
		Kmag: float64(value.kmag),

		ComponentReference: bstringToGoString(value.component_reference),
	}
}

func toLidarReading(lidar string, readings []byte, timestamp time.Time) C.viam_carto_lidar_reading {
	sr := C.viam_carto_lidar_reading{}
	sensorCStr := C.CString(lidar)
	defer C.free(unsafe.Pointer(sensorCStr))
	sr.lidar = C.blk2bstr(unsafe.Pointer(sensorCStr), C.int(len(lidar)))
	readingsCBytes := C.CBytes(readings)
	defer C.free(readingsCBytes)
	sr.lidar_reading = C.blk2bstr(readingsCBytes, C.int(len(readings)))
	sr.lidar_reading_time_unix_milli = C.int64_t(timestamp.UnixMilli())
	return sr
}

func toIMUReading(imu string, readings imuReading, timestamp time.Time) C.viam_carto_imu_reading {
	sr := C.viam_carto_imu_reading{}
	sensorCStr := C.CString(imu)
	defer C.free(unsafe.Pointer(sensorCStr))
	sr.imu = C.blk2bstr(unsafe.Pointer(sensorCStr), C.int(len(imu)))

	sr.lin_acc_x = C.double(readings.LinAccX)
	sr.lin_acc_y = C.double(readings.LinAccY)
	sr.lin_acc_z = C.double(readings.LinAccZ)
	sr.ang_vel_x = C.double(readings.AngVelX)
	sr.ang_vel_y = C.double(readings.AngVelY)
	sr.ang_vel_z = C.double(readings.AngVelZ)

	sr.imu_reading_time_unix_milli = C.int64_t(timestamp.UnixMilli())
	return sr
}

func bstringToByteSlice(bstr C.bstring) []byte {
	return C.GoBytes(unsafe.Pointer(bstr.data), bstr.slen)
}

func toError(status C.int) error {
	switch int(status) {
	case C.VIAM_CARTO_SUCCESS:
		return nil
	case C.VIAM_CARTO_UNABLE_TO_ACQUIRE_LOCK:
		return ErrUnableToAcquireLock
	case C.VIAM_CARTO_VC_INVALID:
		return errors.New("VIAM_CARTO_VC_INVALID")
	case C.VIAM_CARTO_OUT_OF_MEMORY:
		return errors.New("VIAM_CARTO_OUT_OF_MEMORY")
	case C.VIAM_CARTO_DESTRUCTOR_ERROR:
		return errors.New("VIAM_CARTO_DESTRUCTOR_ERROR")
	case C.VIAM_CARTO_LIB_PLATFORM_INVALID:
		return errors.New("VIAM_CARTO_LIB_PLATFORM_INVALID")
	case C.VIAM_CARTO_LIB_INVALID:
		return errors.New("VIAM_CARTO_LIB_INVALID")
	case C.VIAM_CARTO_LIB_NOT_INITIALIZED:
		return errors.New("VIAM_CARTO_LIB_NOT_INITIALIZED")
	case C.VIAM_CARTO_UNKNOWN_ERROR:
		return errors.New("VIAM_CARTO_UNKNOWN_ERROR")
	case C.VIAM_CARTO_DATA_DIR_NOT_PROVIDED:
		return errors.New("VIAM_CARTO_DATA_DIR_NOT_PROVIDED")
	case C.VIAM_CARTO_SLAM_MODE_INVALID:
		return errors.New("VIAM_CARTO_SLAM_MODE_INVALID")
	case C.VIAM_CARTO_LIDAR_CONFIG_INVALID:
		return errors.New("VIAM_CARTO_LIDAR_CONFIG_INVALID")
	case C.VIAM_CARTO_MAP_RATE_SEC_INVALID:
		return errors.New("VIAM_CARTO_MAP_RATE_SEC_INVALID")
	case C.VIAM_CARTO_COMPONENT_REFERENCE_INVALID:
		return errors.New("VIAM_CARTO_COMPONENT_REFERENCE_INVALID")
	case C.VIAM_CARTO_LUA_CONFIG_NOT_FOUND:
		return errors.New("VIAM_CARTO_LUA_CONFIG_NOT_FOUND")
	case C.VIAM_CARTO_DATA_DIR_INVALID_DEPRECATED_STRUCTURE:
		return errors.New("VIAM_CARTO_DATA_DIR_INVALID_DEPRECATED_STRUCTURE")
	case C.VIAM_CARTO_DATA_DIR_FILE_SYSTEM_ERROR:
		return errors.New("VIAM_CARTO_DATA_DIR_FILE_SYSTEM_ERROR")
	case C.VIAM_CARTO_MAP_CREATION_ERROR:
		return errors.New("VIAM_CARTO_MAP_CREATION_ERROR")
	case C.VIAM_CARTO_UNKNOWN_SENSOR_NAME:
		return errors.New("VIAM_CARTO_UNKNOWN_SENSOR_NAME")
	case C.VIAM_CARTO_LIDAR_READING_EMPTY:
		return errors.New("VIAM_CARTO_LIDAR_READING_EMPTY")
	case C.VIAM_CARTO_LIDAR_READING_INVALID:
		return errors.New("VIAM_CARTO_LIDAR_READING_INVALID")
	case C.VIAM_CARTO_GET_POSITION_RESPONSE_INVALID:
		return errors.New("VIAM_CARTO_GET_POSITION_RESPONSE_INVALID")
	case C.VIAM_CARTO_POINTCLOUD_MAP_EMPTY:
		return errors.New("VIAM_CARTO_POINTCLOUD_MAP_EMPTY")
	case C.VIAM_CARTO_GET_POINT_CLOUD_MAP_RESPONSE_INVLALID:
		return errors.New("VIAM_CARTO_GET_POINT_CLOUD_MAP_RESPONSE_INVLALID")
	case C.VIAM_CARTO_LIB_ALREADY_INITIALIZED:
		return errors.New("VIAM_CARTO_LIB_ALREADY_INITIALIZED")
	case C.VIAM_CARTO_GET_INTERNAL_STATE_RESPONSE_INVLALID:
		return errors.New("VIAM_CARTO_GET_INTERNAL_STATE_RESPONSE_INVLALID")
	case C.VIAM_CARTO_GET_INTERNAL_STATE_FILE_WRITE_IO_ERROR:
		return errors.New("VIAM_CARTO_GET_INTERNAL_STATE_FILE_WRITE_IO_ERROR")
	case C.VIAM_CARTO_GET_INTERNAL_STATE_FILE_READ_IO_ERROR:
		return errors.New("VIAM_CARTO_GET_INTERNAL_STATE_FILE_READ_IO_ERROR")
	case C.VIAM_CARTO_NOT_IN_INITIALIZED_STATE:
		return errors.New("VIAM_CARTO_NOT_IN_INITIALIZED_STATE")
	case C.VIAM_CARTO_NOT_IN_IO_INITIALIZED_STATE:
		return errors.New("VIAM_CARTO_NOT_IN_IO_INITIALIZED_STATE")
	case C.VIAM_CARTO_NOT_IN_STARTED_STATE:
		return errors.New("VIAM_CARTO_NOT_IN_STARTED_STATE")
	case C.VIAM_CARTO_NOT_IN_TERMINATABLE_STATE:
		return errors.New("VIAM_CARTO_NOT_IN_TERMINATABLE_STATE")
	case C.VIAM_CARTO_IMU_CONFIG_INVALID:
		return errors.New("VIAM_CARTO_IMU_CONFIG_INVALID")
	case C.VIAM_CARTO_IMU_READING_EMPTY:
		return errors.New("VIAM_CARTO_IMU_READING_EMPTY")
	case C.VIAM_CARTO_IMU_READING_INVALID:
		return errors.New("VIAM_CARTO_IMU_READING_INVALID")
	default:
		return errors.New("status code unclassified")
	}
}
