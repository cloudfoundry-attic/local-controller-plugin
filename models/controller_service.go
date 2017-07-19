package models

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"code.cloudfoundry.org/goshims/filepathshim"
	"code.cloudfoundry.org/goshims/osshim"
	"code.cloudfoundry.org/lager"
	. "github.com/jeffpak/csi"
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
		return &CreateVolumeResponse{}, errors.New("Missing mandatory 'volume_name'")
	}

	if _, ok = cs.volumes[volName]; !ok {
		logger.Info("creating-volume", lager.Data{"volume_name": volName, "volume_id": volName})
		localVol := LocalVolume{VolumeInfo: VolumeInfo{Id: &VolumeID{Values: map[string]string{"volume_name": volName}}, AccessMode: &AccessMode{Mode: AccessMode_UNKNOWN}}}
		cs.volumes[in.Name] = &localVol

		resp := &CreateVolumeResponse{Reply: &CreateVolumeResponse_Result_{
			Result: &CreateVolumeResponse_Result{
				VolumeInfo: &localVol.VolumeInfo,
			}}}

		logger.Info("CreateVolumeResponse", lager.Data{"resp": resp})
		return resp, nil
	}
	return &CreateVolumeResponse{}, errors.New("should not have gotten here!!!")
}

func (cs *Controller) DeleteVolume(context context.Context, request *DeleteVolumeRequest) (*DeleteVolumeResponse, error) {
	logger := cs.logger.Session("delete-volume")
	logger.Info("start")
	defer logger.Info("end")

	var volName string
	var ok bool

	if volName, ok = request.GetVolumeId().GetValues()["volume_name"]; !ok {
		logger.Error("failed-volume-deletion", fmt.Errorf("Request has no volume name"))
		return &DeleteVolumeResponse{}, errors.New("Request missing 'volume_name'")
	}
	if volName == "" {
		logger.Error("failed-volume-deletion", fmt.Errorf("Request has blank volume name"))
		return &DeleteVolumeResponse{}, errors.New("Request needs non-empty 'volume_name'")
	}

	if _, exists := cs.volumes[volName]; !exists {
		logger.Error("failed-volume-removal", errors.New(fmt.Sprintf("Volume %s not found", volName)))
		return &DeleteVolumeResponse{}, errors.New(fmt.Sprintf("Volume '%s' not found", volName))
	}
	return &DeleteVolumeResponse{Reply: &DeleteVolumeResponse_Result_{
		Result: &DeleteVolumeResponse_Result{},
	}}, nil

}
func (cs *Controller) ControllerPublishVolume(ctx context.Context, in *ControllerPublishVolumeRequest) (*ControllerPublishVolumeResponse, error) {
	return &ControllerPublishVolumeResponse{}, nil
}

func (cs *Controller) ControllerUnpublishVolume(ctx context.Context, in *ControllerUnpublishVolumeRequest) (*ControllerUnpublishVolumeResponse, error) {
	return &ControllerUnpublishVolumeResponse{}, nil
}

func (cs *Controller) ValidateVolumeCapabilities(ctx context.Context, in *ValidateVolumeCapabilitiesRequest) (*ValidateVolumeCapabilitiesResponse, error) {
  for _, vc := range in.GetVolumeCapabilities() {
    if vc.GetMount().GetFsType() != "" {
      return &ValidateVolumeCapabilitiesResponse{
        Reply: &ValidateVolumeCapabilitiesResponse_Result_{
          Result: &ValidateVolumeCapabilitiesResponse_Result{
            Supported: false,
            Message: "Specifying FsType is unsupported.",
          }}}, nil
    }
    for _, flag := range vc.GetMount().GetMountFlags() {
      if flag != "" {
        return &ValidateVolumeCapabilitiesResponse{
          Reply: &ValidateVolumeCapabilitiesResponse_Result_{
            Result: &ValidateVolumeCapabilitiesResponse_Result{
              Supported: false,
              Message: "Specifying mount flags is unsupported.",
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
	return &ListVolumesResponse{}, nil
}

func (cs *Controller) GetCapacity(ctx context.Context, in *GetCapacityRequest) (*GetCapacityResponse, error) {
	return &GetCapacityResponse{}, nil
}

func (cs *Controller) ControllerGetCapabilities(ctx context.Context, in *ControllerGetCapabilitiesRequest) (*ControllerGetCapabilitiesResponse, error) {
	return &ControllerGetCapabilitiesResponse{Reply: &ControllerGetCapabilitiesResponse_Result_{
		Result: &ControllerGetCapabilitiesResponse_Result{
			Capabilities: []*ControllerServiceCapability{{
				Type: &ControllerServiceCapability_Rpc{
					Rpc: &ControllerServiceCapability_RPC{
						Type: ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME}}}}}}}, nil
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
