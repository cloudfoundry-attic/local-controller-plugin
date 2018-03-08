package controller

import (
	"os"
	"path/filepath"
	"syscall"

	"code.cloudfoundry.org/goshims/filepathshim"
	"code.cloudfoundry.org/goshims/osshim"
	"code.cloudfoundry.org/lager"
	. "github.com/container-storage-interface/spec/lib/go/csi/v0"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

const VolumesRootDir = "_volumes"
const MountsRootDir = "_mounts"

type LocalVolume struct {
	Volume
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

	var volId string = in.GetName()
	var ok bool
	if volId == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "Volume name not supplied")
	}

	var localVol *LocalVolume

	logger.Info("creating-volume", lager.Data{"volume_name": volId, "volume_id": volId})

	if _, ok = cs.volumes[volId]; !ok {
		localVol = &LocalVolume{Volume: Volume{Id: volId}}
		cs.volumes[in.Name] = localVol
	}
	localVol = cs.volumes[volId]

	resp := &CreateVolumeResponse{
		Volume: &localVol.Volume,
	}

	logger.Info("CreateVolumeResponse", lager.Data{"resp": resp})
	return resp, nil
}

func (cs *Controller) DeleteVolume(context context.Context, request *DeleteVolumeRequest) (*DeleteVolumeResponse, error) {
	logger := cs.logger.Session("delete-volume")
	logger.Info("start")
	defer logger.Info("end")

	volId := request.GetVolumeId()
	if volId == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "Volume name not supplied")
	}

	delete(cs.volumes, volId)

	return &DeleteVolumeResponse{}, nil
}

func (cs *Controller) ControllerPublishVolume(ctx context.Context, in *ControllerPublishVolumeRequest) (*ControllerPublishVolumeResponse, error) {
	return &ControllerPublishVolumeResponse{PublishInfo: map[string]string{}}, nil
}

func (cs *Controller) ControllerUnpublishVolume(ctx context.Context, in *ControllerUnpublishVolumeRequest) (*ControllerUnpublishVolumeResponse, error) {
	return &ControllerUnpublishVolumeResponse{}, nil
}

func (cs *Controller) ValidateVolumeCapabilities(ctx context.Context, in *ValidateVolumeCapabilitiesRequest) (*ValidateVolumeCapabilitiesResponse, error) {
	for _, vc := range in.GetVolumeCapabilities() {
		if vc.GetMount().GetFsType() != "" {
			return &ValidateVolumeCapabilitiesResponse{
				Supported: false,
				Message:   "Specifying FsType is unsupported.",
			}, nil
		}
		for _, flag := range vc.GetMount().GetMountFlags() {
			if flag != "" {
				return &ValidateVolumeCapabilitiesResponse{
					Supported: false,
					Message:   "Specifying mount flags is unsupported.",
				}, nil
			}
		}
	}
	return &ValidateVolumeCapabilitiesResponse{
		Supported: true,
	}, nil
}

func (cs *Controller) ListVolumes(ctx context.Context, in *ListVolumesRequest) (*ListVolumesResponse, error) {
	var volList []*ListVolumesResponse_Entry

	for _, v := range cs.volumes {
		entry := &ListVolumesResponse_Entry{
			Volume: &v.Volume,
		}
		volList = append(volList, entry)
	}

	return &ListVolumesResponse{
		Entries: volList,
	}, nil
}

func (cs *Controller) GetPluginCapabilities(ctx context.Context, in *GetPluginCapabilitiesRequest) (*GetPluginCapabilitiesResponse, error) {
	return &GetPluginCapabilitiesResponse{Capabilities: []*PluginCapability{}}, nil
}

func (cs *Controller) GetCapacity(ctx context.Context, in *GetCapacityRequest) (*GetCapacityResponse, error) {
	return &GetCapacityResponse{
		AvailableCapacity: ^int64(0),
	}, nil
}

func (cs *Controller) Probe(ctx context.Context, in *ProbeRequest) (*ProbeResponse, error) {
	return &ProbeResponse{}, nil
}

func (cs *Controller) ControllerGetCapabilities(ctx context.Context, in *ControllerGetCapabilitiesRequest) (*ControllerGetCapabilitiesResponse, error) {
	return &ControllerGetCapabilitiesResponse{
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
	}, nil
}

func (cs *Controller) GetPluginInfo(ctx context.Context, in *GetPluginInfoRequest) (*GetPluginInfoResponse, error) {
	return &GetPluginInfoResponse{
		Name:          "com.github.jeffpak.local-controller-plugin",
		VendorVersion: "0.1.0",
	}, nil
}

func (cs *Controller) volumePath(logger lager.Logger, volumeId string) string {
	dir, err := cs.filepath.Abs(cs.mountPathRoot)
	if err != nil {
		logger.Fatal("abs-failed", err)
	}

	volumesPathRoot := filepath.Join(dir, VolumesRootDir)
	orig := syscall.Umask(000)
	defer syscall.Umask(orig)
	err = cs.os.MkdirAll(volumesPathRoot, os.ModePerm)

	if err != nil {
		logger.Fatal("mkdir-all-failed", err)
	}

	return filepath.Join(volumesPathRoot, volumeId)
}
