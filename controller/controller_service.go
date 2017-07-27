package controller

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"code.cloudfoundry.org/goshims/filepathshim"
	"code.cloudfoundry.org/goshims/osshim"
	"code.cloudfoundry.org/lager"
	. "github.com/container-storage-interface/spec"
	"golang.org/x/net/context"
)

const VolumesRootDir = "_volumes"
const MountsRootDir = "_mounts"

type LocalVolume struct {
	VolumeInfo
}

type Controller struct {
	logger   lager.Logger
	volumes  map[string]*LocalVolume
	os       osshim.Os
	filepath filepathshim.Filepath
	///Where does this fit into the create/mount logic?
	mountPathRoot string
}

func NewController(osshim osshim.Os, filepath filepathshim.Filepath, mountPathRoot string) *Controller {
	logger := lager.NewLogger("local-controller-plugin")
	sink := lager.NewReconfigurableSink(lager.NewWriterSink(os.Stdout, lager.DEBUG), lager.DEBUG)
	logger.RegisterSink(sink)

	return &Controller{
		logger:        logger,
		volumes:       map[string]*LocalVolume{},
		os:            osshim,
		filepath:      filepath,
		mountPathRoot: mountPathRoot,
	}
}

func (cs *Controller) CreateVolume(ctx context.Context, in *CreateVolumeRequest) (*CreateVolumeResponse, error) {
	logger := cs.logger.Session("create-volume")
	logger.Info("start")
	defer logger.Info("end")

	var volName string = in.GetName()
	var ok bool
	if volName == "" {
		return createCreateVolumeErrorResponse(Error_CreateVolumeError_INVALID_VOLUME_NAME, "Volume name not supplied"), nil
	}

	var localVol *LocalVolume

	logger.Info("creating-volume", lager.Data{"volume_name": volName, "volume_id": volName})

	if _, ok = cs.volumes[volName]; !ok {
		localVol = &LocalVolume{VolumeInfo: VolumeInfo{Id: &VolumeID{Values: map[string]string{"volume_name": volName}}, AccessMode: &AccessMode{Mode: AccessMode_UNKNOWN}}}
		cs.volumes[in.Name] = localVol
	}
	localVol = cs.volumes[volName]

	resp := &CreateVolumeResponse{Reply: &CreateVolumeResponse_Result_{
		Result: &CreateVolumeResponse_Result{
			VolumeInfo: &localVol.VolumeInfo,
		}}}

	logger.Info("CreateVolumeResponse", lager.Data{"resp": resp})
	return resp, nil
}

func (cs *Controller) DeleteVolume(context context.Context, request *DeleteVolumeRequest) (*DeleteVolumeResponse, error) {
	logger := cs.logger.Session("delete-volume")
	logger.Info("start")
	defer logger.Info("end")

	var volName, errorDescription string
	var ok bool

	if volName, ok = request.GetVolumeId().GetValues()["volume_name"]; !ok {
		errorDescription = "Request missing 'volume_name'"
		logger.Error("failed-volume-deletion", fmt.Errorf(errorDescription))
		return createDeleteVolumeErrorResponse(Error_DeleteVolumeError_INVALID_VOLUME_ID, errorDescription), nil
	}
	if volName == "" {
		errorDescription = "Request has blank volume name"
		logger.Error("failed-volume-deletion", fmt.Errorf(errorDescription))
		return createDeleteVolumeErrorResponse(Error_DeleteVolumeError_INVALID_VOLUME_ID, errorDescription), nil
	}

	delete(cs.volumes, volName)

	return &DeleteVolumeResponse{Reply: &DeleteVolumeResponse_Result_{
		Result: &DeleteVolumeResponse_Result{},
	}}, nil

}
func (cs *Controller) ControllerPublishVolume(ctx context.Context, in *ControllerPublishVolumeRequest) (*ControllerPublishVolumeResponse, error) {
	return &ControllerPublishVolumeResponse{Reply: &ControllerPublishVolumeResponse_Result_{
		Result: &ControllerPublishVolumeResponse_Result{
			PublishVolumeInfo: &PublishVolumeInfo{},
		},
	}}, nil
}

func (cs *Controller) ControllerUnpublishVolume(ctx context.Context, in *ControllerUnpublishVolumeRequest) (*ControllerUnpublishVolumeResponse, error) {
	return &ControllerUnpublishVolumeResponse{Reply: &ControllerUnpublishVolumeResponse_Result_{
		Result: &ControllerUnpublishVolumeResponse_Result{},
	}}, nil
}

func (cs *Controller) ValidateVolumeCapabilities(ctx context.Context, in *ValidateVolumeCapabilitiesRequest) (*ValidateVolumeCapabilitiesResponse, error) {
	for _, vc := range in.GetVolumeCapabilities() {
		if vc.GetMount().GetFsType() != "" {
			return &ValidateVolumeCapabilitiesResponse{
				Reply: &ValidateVolumeCapabilitiesResponse_Result_{
					Result: &ValidateVolumeCapabilitiesResponse_Result{
						Supported: false,
						Message:   "Specifying FsType is unsupported.",
					}}}, nil
		}
		for _, flag := range vc.GetMount().GetMountFlags() {
			if flag != "" {
				return &ValidateVolumeCapabilitiesResponse{
					Reply: &ValidateVolumeCapabilitiesResponse_Result_{
						Result: &ValidateVolumeCapabilitiesResponse_Result{
							Supported: false,
							Message:   "Specifying mount flags is unsupported.",
						}}}, nil
			}
		}
	}
	return &ValidateVolumeCapabilitiesResponse{
		Reply: &ValidateVolumeCapabilitiesResponse_Result_{
			Result: &ValidateVolumeCapabilitiesResponse_Result{
				Supported: true,
			}}}, nil
}

func (cs *Controller) ListVolumes(ctx context.Context, in *ListVolumesRequest) (*ListVolumesResponse, error) {
	var volList []*ListVolumesResponse_Result_Entry

	for _, v := range cs.volumes {
		entry := &ListVolumesResponse_Result_Entry{
			VolumeInfo: &v.VolumeInfo,
		}
		volList = append(volList, entry)
	}

	return &ListVolumesResponse{
		Reply: &ListVolumesResponse_Result_{
			Result: &ListVolumesResponse_Result{
				Entries: volList,
			},
		},
	}, nil
}

func (cs *Controller) GetCapacity(ctx context.Context, in *GetCapacityRequest) (*GetCapacityResponse, error) {
	return &GetCapacityResponse{
		Reply: &GetCapacityResponse_Result_{
			Result: &GetCapacityResponse_Result{
				TotalCapacity: ^uint64(0),
			},
		},
	}, nil
}

func (cs *Controller) ControllerGetCapabilities(ctx context.Context, in *ControllerGetCapabilitiesRequest) (*ControllerGetCapabilitiesResponse, error) {
	return &ControllerGetCapabilitiesResponse{Reply: &ControllerGetCapabilitiesResponse_Result_{
		Result: &ControllerGetCapabilitiesResponse_Result{
			Capabilities: []*ControllerServiceCapability{
				{
					Type: &ControllerServiceCapability_Rpc{
						Rpc: &ControllerServiceCapability_RPC{
							Type: ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME,
						},
					},
				},
				{
					Type: &ControllerServiceCapability_Rpc{
						Rpc: &ControllerServiceCapability_RPC{
							Type: ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME,
						},
					},
				},
				{
					Type: &ControllerServiceCapability_Rpc{
						Rpc: &ControllerServiceCapability_RPC{
							Type: ControllerServiceCapability_RPC_LIST_VOLUMES,
						},
					},
				},
				{
					Type: &ControllerServiceCapability_Rpc{
						Rpc: &ControllerServiceCapability_RPC{
							Type: ControllerServiceCapability_RPC_GET_CAPACITY,
						},
					},
				},
			},
		}}}, nil
}

func (cs *Controller) volumePath(logger lager.Logger, volumeId string) string {
	dir, err := cs.filepath.Abs(cs.mountPathRoot)
	if err != nil {
		logger.Fatal("abs-failed", err)
	}

	volumesPathRoot := filepath.Join(dir, VolumesRootDir)
	orig := syscall.Umask(000)
	defer syscall.Umask(orig)
	cs.os.MkdirAll(volumesPathRoot, os.ModePerm)

	return filepath.Join(volumesPathRoot, volumeId)
}

func createCreateVolumeErrorResponse(errorCode Error_CreateVolumeError_CreateVolumeErrorCode, errorDescription string) *CreateVolumeResponse {
	return &CreateVolumeResponse{
		Reply: &CreateVolumeResponse_Error{
			Error: &Error{
				Value: &Error_CreateVolumeError_{
					CreateVolumeError: &Error_CreateVolumeError{
						ErrorCode:        errorCode,
						ErrorDescription: errorDescription,
					}}}}}
}

func createDeleteVolumeErrorResponse(errorCode Error_DeleteVolumeError_DeleteVolumeErrorCode, errorDescription string) *DeleteVolumeResponse {
	return &DeleteVolumeResponse{
		Reply: &DeleteVolumeResponse_Error{
			Error: &Error{
				Value: &Error_DeleteVolumeError_{
					DeleteVolumeError: &Error_DeleteVolumeError{
						ErrorCode:        errorCode,
						ErrorDescription: errorDescription,
					}}}}}
}
