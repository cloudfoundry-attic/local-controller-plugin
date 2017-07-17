package models

import(
  "github.com/container-storage-interface/csi"
  "golang.org/x/net/context"
  "code.cloudfoundry.org/goshims/osshim"
  "code.cloudfoundry.org/goshims/filepathshim"
  "code.cloudfoundry.org/lager"
  "syscall"
  "os"
  "path/filepath"
  "errors"
  "fmt"
)

const VolumesRootDir = "_volumes"
const MountsRootDir = "_mounts"

type LocalVolume struct {
  VolumeInfo
}

type Controller struct {
  volumes       map[string]*LocalVolume
  os            osshim.Os
  filepath      filepathshim.Filepath
  ///Where does this fit into the create/mount logic?
  mountPathRoot string
}

func NewController(os osshim.Os, filepath filepathshim.Filepath, mountPathRoot string) *Controller {
  return &Controller{
    volumes:       map[string]*LocalVolume{},
    os:            os,
    filepath:      filepath,
    mountPathRoot: mountPathRoot,
  }
}

func (cs *Controller) CreateVolume(ctx context.Context, in *CreateVolumeRequest) (*CreateVolumeResponse, error) {
  logger := lager.NewLogger("create-volume")
  var volName string = in.GetName()
  var ok bool
  if volName == "" {
    return &CreateVolumeResponse{}, errors.New("Missing mandatory 'volume_name'")
  }

  //var existingVolume *VolumeInfo
  if _, ok = cs.volumes[volName]; !ok {
    logger.Info("creating-volume", lager.Data{"volume_name": volName, "volume_id": volName})
    localVol := LocalVolume{VolumeInfo: VolumeInfo{Id: &VolumeID{Values: map[string]string{"volume_name": volName}}, AccessMode: &AccessMode{Mode: AccessMode_UNKNOWN}}}
    cs.volumes[in.Name] = &localVol

    return &CreateVolumeResponse{Reply: &CreateVolumeResponse_Result_{
      Result: &CreateVolumeResponse_Result{
        VolumeInfo: &localVol.VolumeInfo,
      }}}, nil
  }
  //TODO: Is this ever going to happen? Won't Name and Id always be the same?
  //if existingVolume.Id != in.Name {
    //logger.Info("duplicate-volume", lager.Data{"volume_name": in.Name})
  //  return CreateVolumeResponse{}, errors.New(fmt.Sprintf("Volume '%s' already exists with a different volume ID", in.Name))
  //}

  return &CreateVolumeResponse{}, nil
}

func (cs *Controller) DeleteVolume(context context.Context, request *DeleteVolumeRequest) (*DeleteVolumeResponse, error) {
  logger := lager.NewLogger("delete-volume")
  logger.Info("start")
  defer logger.Info("end")
  var volName string
  var ok bool

  if volName, ok = request.GetVolumeId().GetValues()["volume_name"]; !ok {
    //logger.Error("failed-volume-deletion", fmt.Errorf("Request has no volume name"))
    //return &DeleteVolumeResponse, errors.New("Request missing volume name")
  }


  if volName == "" {
    return &DeleteVolumeResponse{}, errors.New("Missing mandatory 'volume_name'")
  }

  //var vol *LocalVolume
  //var exists bool
  if _, exists := cs.volumes[volName]; !exists {
    logger.Error("failed-volume-removal", errors.New(fmt.Sprintf("Volume %s not found", volName)))
    return &DeleteVolumeResponse{}, errors.New(fmt.Sprintf("Volume '%s' not found", volName))
  }

  //TODO: Mountpoint logic
  //if vol.Mountpoint != "" {
  //  response = d.unmount(logger, removeRequest.Name, vol.Mountpoint)
  //  if response.Err != "" {
  //    return response
  //  }
  //}

  volumePath := cs.volumePath(logger, volName)

  logger.Info("remove-volume-folder", lager.Data{"volume": volumePath})
  err := cs.os.RemoveAll(volumePath)
  if err != nil {
    logger.Error("failed-removing-volume", err)
    return &DeleteVolumeResponse{}, errors.New(fmt.Sprintf("Failed removing mount path: %s", err))
  }

  logger.Info("removing-volume", lager.Data{"name": volName})
  delete(cs.volumes, volName)
  return &DeleteVolumeResponse{}, nil

}
func (cs *Controller) ControllerPublishVolume(ctx context.Context, in *ControllerPublishVolumeRequest) (*ControllerPublishVolumeResponse, error) { return &ControllerPublishVolumeResponse{}, nil }

func (cs *Controller) ControllerUnpublishVolume(ctx context.Context, in *ControllerUnpublishVolumeRequest) (*ControllerUnpublishVolumeResponse, error) { return &ControllerUnpublishVolumeResponse{}, nil }

func (cs *Controller) ValidateVolumeCapabilities(ctx context.Context, in *ValidateVolumeCapabilitiesRequest) (*ValidateVolumeCapabilitiesResponse, error) { return &ValidateVolumeCapabilitiesResponse{}, nil }

func (cs *Controller) ListVolumes(ctx context.Context, in *ListVolumesRequest) (*ListVolumesResponse, error) {return &ListVolumesResponse{}, nil }

func (cs *Controller) GetCapacity(ctx context.Context, in *GetCapacityRequest) (*GetCapacityResponse, error) { return &GetCapacityResponse{}, nil }

func (cs *Controller) ControllerGetCapabilities(ctx context.Context, in *ControllerGetCapabilitiesRequest) (*ControllerGetCapabilitiesResponse, error) { return &ControllerGetCapabilitiesResponse{}, nil}


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
