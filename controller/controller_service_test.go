package controller_test

import (
	"time"

	"code.cloudfoundry.org/goshims/filepathshim/filepath_fake"
	"code.cloudfoundry.org/goshims/osshim/os_fake"
	. "github.com/container-storage-interface/spec/lib/go/csi/v0"
	"github.com/jeffpak/local-controller-plugin/controller"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/types"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func VolumeIDMatcher(volID string) GomegaMatcher {
	return WithTransform(func(entry *ListVolumesResponse_Entry) string {
		return entry.GetVolume().GetId()
	}, Equal(volID))
}

var _ = Describe("ControllerService", func() {
	var (
		cs      *controller.Controller
		context context.Context

		fakeOs       *os_fake.FakeOs
		fakeFilepath *filepath_fake.FakeFilepath
		mountDir     string
		volumeName   string
		volumeId     string
		vc           []*VolumeCapability
		vol          *Volume
		err          error
	)

	BeforeEach(func() {
		mountDir = "/path/to/mount"
		fakeOs = &os_fake.FakeOs{}
		fakeFilepath = &filepath_fake.FakeFilepath{}
		cs = controller.NewController(fakeOs, fakeFilepath, mountDir)
		context = &DummyContext{}
		volumeId = "vol-name"
		volumeName = "vol-name"
		vc = []*VolumeCapability{{AccessType: &VolumeCapability_Mount{Mount: &VolumeCapability_MountVolume{}}}}
		vol = &Volume{Id: volumeId}
	})

	Describe("CreateVolume", func() {
		var (
			expectedResponse *CreateVolumeResponse
		)

		BeforeEach(func() {
			expectedResponse = createSuccessful(context, cs, fakeOs, volumeName, vc)
		})

		It("does not fail", func() {
			Expect(*expectedResponse).To(Equal(CreateVolumeResponse{
				Volume: vol,
			}))
		})

		Context("when the Volume exists", func() {
			BeforeEach(func() {
				expectedResponse = createSuccessful(context, cs, fakeOs, volumeName, vc)
			})

			It("should succeed and respond with the existent volume", func() {
				Expect(*expectedResponse).To(Equal(CreateVolumeResponse{
					Volume: vol,
				}))
			})
		})

		Context("when the request is invalid (no volume name)", func() {
			var (
				err            error
				createVolReq   *CreateVolumeRequest
				createResponse *CreateVolumeResponse
			)
			BeforeEach(func() {
				createVolReq = &CreateVolumeRequest{
					Name:               "",
					VolumeCapabilities: vc,
				}
			})
			JustBeforeEach(func() {
				createResponse, err = cs.CreateVolume(context, createVolReq)
			})
			It("should fail with an error response", func() {
				Expect(createResponse).To(BeNil())
				Expect(err).To(HaveOccurred())
				grpcStatus, _ := status.FromError(err)
				Expect(grpcStatus.Code()).To(Equal(codes.InvalidArgument))
			})
		})

		Describe("DeleteVolume", func() {
			var (
				deleteVolResponse *DeleteVolumeResponse

				listReq  *ListVolumesRequest
				listResp *ListVolumesResponse
			)

			It("should fail if no volume ID is provided in the request", func() {
				deleteVolResponse, err = cs.DeleteVolume(context, &DeleteVolumeRequest{})
				Expect(deleteVolResponse).To(BeNil())
				Expect(err).To(HaveOccurred())
				grpcStatus, _ := status.FromError(err)
				Expect(grpcStatus.Code()).To(Equal(codes.InvalidArgument))
			})

			It("should succeed if no volume was found", func() {
				deleteVolResponse, err = cs.DeleteVolume(context, &DeleteVolumeRequest{
					VolumeId: "non-existent-volume",
				})
				Expect(err).To(BeNil())
			})

			Context("when the volume has been created", func() {
				var (
					createVolResponse *CreateVolumeResponse
				)
				BeforeEach(func() {
					createVolResponse = createSuccessful(context, cs, fakeOs, volumeName, vc)
				})

				It("should delete the volume", func() {
					response := deleteSuccessful(context, cs, volumeId)
					Expect(response).NotTo(BeNil())

					listReq = &ListVolumesRequest{
						MaxEntries: 100,
					}

					listResp, err = cs.ListVolumes(context, listReq)
					Expect(err).NotTo(HaveOccurred())
					Expect(listResp).NotTo(BeNil())
					volID := createVolResponse.GetVolume().GetId()
					Expect(listResp.GetEntries()).NotTo(ContainElement(VolumeIDMatcher(volID)))
				})
			})
		})

		Describe("ControllerPublishVolume", func() {
			var (
				request          *ControllerPublishVolumeRequest
				expectedResponse *ControllerPublishVolumeResponse
			)

			Context("when ControllerPublishVolume is called with a ControllerPublishVolumeRequest", func() {
				BeforeEach(func() {
					request = &ControllerPublishVolumeRequest{
						VolumeId:                 volumeId,
						NodeId:                   "",
						VolumeCapability:         vc[0],
						Readonly:                 false,
						ControllerPublishSecrets: nil,
						VolumeAttributes:         nil,
					}
				})
				JustBeforeEach(func() {
					expectedResponse, err = cs.ControllerPublishVolume(context, request)
				})
				It("should return a ControllerPublishVolumeResponse", func() {
					Expect(err).ToNot(HaveOccurred())
					Expect(expectedResponse).NotTo(BeNil())
					Expect(expectedResponse.GetPublishInfo()).NotTo(BeNil())
				})
			})
		})

		Describe("ControllerUnpublishVolume", func() {
			var (
				request          *ControllerUnpublishVolumeRequest
				expectedResponse *ControllerUnpublishVolumeResponse
			)
			Context("when ControllerUnpublishVolume is called with a ControllerUnpublishVolumeRequest", func() {
				BeforeEach(func() {
					request = &ControllerUnpublishVolumeRequest{
						VolumeId: volumeId,
						NodeId:   "",
						ControllerUnpublishSecrets: map[string]string{},
					}
				})
				JustBeforeEach(func() {
					expectedResponse, err = cs.ControllerUnpublishVolume(context, request)
				})
				It("should return a ControllerUnpublishVolumeResponse", func() {
					Expect(err).ToNot(HaveOccurred())
					Expect(expectedResponse).NotTo(BeNil())
				})
				Context("when a volume is unpublished for a second time", func() {
					JustBeforeEach(func() {
						expectedResponse, err = cs.ControllerUnpublishVolume(context, request)
					})
					It("should return a ControllerUnpublishVolumeResponse", func() {
						Expect(err).ToNot(HaveOccurred())
						Expect(expectedResponse).NotTo(BeNil())
					})
				})
			})
		})

		Describe("ValidateVolumeCapabilities", func() {
			var (
				request          *ValidateVolumeCapabilitiesRequest
				expectedResponse *ValidateVolumeCapabilitiesResponse
			)
			Context("when called with no capabilities", func() {
				BeforeEach(func() {
					request = &ValidateVolumeCapabilitiesRequest{
						VolumeId: volumeId,
						VolumeCapabilities: []*VolumeCapability{{AccessType: &VolumeCapability_Mount{
							Mount: &VolumeCapability_MountVolume{
								MountFlags: []string{""},
							}}}},
						VolumeAttributes: nil,
					}
				})
				JustBeforeEach(func() {
					expectedResponse, err = cs.ValidateVolumeCapabilities(context, request)
				})
				It("should return a ValidateVolumeResponse", func() {
					Expect(err).To(BeNil())
					Expect(expectedResponse).To(Equal(&ValidateVolumeCapabilitiesResponse{Supported: true}))
				})
			})

			Context("when called with unsupported FsType capabilities", func() {
				BeforeEach(func() {
					request = &ValidateVolumeCapabilitiesRequest{
						VolumeId: volumeId,
						VolumeCapabilities: []*VolumeCapability{{AccessType: &VolumeCapability_Mount{
							Mount: &VolumeCapability_MountVolume{
								FsType: "unsupported",
							}}}},
					}
				})
				JustBeforeEach(func() {
					expectedResponse, err = cs.ValidateVolumeCapabilities(context, request)
				})
				It("should return an error", func() {
					Expect(err).To(BeNil())
					Expect(expectedResponse).To(Equal(&ValidateVolumeCapabilitiesResponse{
						Supported: false,
						Message:   "Specifying FsType is unsupported.",
					}))
				})
			})

			Context("when called with unsupported MountFlag capabilities", func() {
				BeforeEach(func() {
					request = &ValidateVolumeCapabilitiesRequest{
						VolumeId: volumeId,
						VolumeCapabilities: []*VolumeCapability{{AccessType: &VolumeCapability_Mount{
							Mount: &VolumeCapability_MountVolume{
								MountFlags: []string{"unsupported"},
							}}}},
					}
				})
				JustBeforeEach(func() {
					expectedResponse, err = cs.ValidateVolumeCapabilities(context, request)
				})
				It("should return an error", func() {
					Expect(err).To(BeNil())
					Expect(expectedResponse).To(Equal(&ValidateVolumeCapabilitiesResponse{
						Supported: false,
						Message:   "Specifying mount flags is unsupported.",
					}))
				})
			})
		})

		Describe("when volumes are listed", func() {
			var (
				request          *ListVolumesRequest
				expectedResponse *ListVolumesResponse
			)

			JustBeforeEach(func() {
				request = &ListVolumesRequest{
					MaxEntries:    10,
					StartingToken: "starting-token",
				}
				expectedResponse, err = cs.ListVolumes(context, request)
			})

			It("should return a response listing that volume", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(expectedResponse).NotTo(BeNil())
				Expect(expectedResponse.GetEntries()).To(ContainElement(VolumeIDMatcher(volumeId)))
			})
		})

		Describe("ControllerProbe", func() {
			var (
				request          *ProbeRequest
				expectedResponse *ProbeResponse
			)
			BeforeEach(func() {
				request = &ProbeRequest{}
			})

			JustBeforeEach(func() {
				expectedResponse, _ = cs.Probe(context, request)
			})

			It("should return a ControllerProbeResponse", func() {
				Expect(*expectedResponse).NotTo(BeNil())
				Expect(expectedResponse).ToNot(BeNil())
			})
		})

		Describe("GetCapacity", func() {
			var (
				request          *GetCapacityRequest
				expectedResponse *GetCapacityResponse
			)
			Context("when GetCapacity is called with a GetCapacityRequest", func() {
				BeforeEach(func() {
					request = &GetCapacityRequest{
						VolumeCapabilities: vc,
						Parameters:         map[string]string{},
						AccessibleTopology: nil,
					}
				})
				JustBeforeEach(func() {
					expectedResponse, err = cs.GetCapacity(context, request)
				})
				It("should return a GetCapacityResponse", func() {
					Expect(err).ToNot(HaveOccurred())
					Expect(expectedResponse).NotTo(BeNil())
					Expect(expectedResponse.GetAvailableCapacity()).NotTo(BeNil())
				})
			})
		})

		Describe("ControllerGetCapabilities", func() {
			var (
				request          *ControllerGetCapabilitiesRequest
				expectedResponse *ControllerGetCapabilitiesResponse
			)
			Context("when ControllerGetCapabilities is called with a ControllerGetCapabilitiesRequest", func() {
				BeforeEach(func() {
					request = &ControllerGetCapabilitiesRequest{}
				})
				JustBeforeEach(func() {
					expectedResponse, err = cs.ControllerGetCapabilities(context, request)
				})

				It("should return a listing all capabilities", func() {
					Expect(expectedResponse).NotTo(BeNil())
					capabilities := expectedResponse.GetCapabilities()
					Expect(capabilities).To(HaveLen(4))
					Expect(capabilities[0].GetRpc().GetType()).To(Equal(ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME))
					Expect(capabilities[1].GetRpc().GetType()).To(Equal(ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME))
					Expect(capabilities[2].GetRpc().GetType()).To(Equal(ControllerServiceCapability_RPC_LIST_VOLUMES))
					Expect(capabilities[3].GetRpc().GetType()).To(Equal(ControllerServiceCapability_RPC_GET_CAPACITY))
				})
			})
		})
	})

	Describe("GetPluginInfo", func() {
		var (
			request          *GetPluginInfoRequest
			expectedResponse *GetPluginInfoResponse
		)
		Context("when provided with a GetPluginInfoRequest", func() {
			BeforeEach(func() {
				request = &GetPluginInfoRequest{}
			})

			JustBeforeEach(func() {
				expectedResponse, err = cs.GetPluginInfo(context, request)
			})

			It("returns the plugin info", func() {
				Expect(expectedResponse).NotTo(BeNil())
				Expect(err).ToNot(HaveOccurred())
				Expect(expectedResponse.GetName()).To(Equal("com.github.jeffpak.local-controller-plugin"))
				Expect(expectedResponse.GetVendorVersion()).To(Equal("0.1.0"))
			})
		})
	})

	Describe("GetPluginCapabilities", func() {
		var (
			request          *GetPluginCapabilitiesRequest
			expectedResponse *GetPluginCapabilitiesResponse
		)

		BeforeEach(func() {
			request = &GetPluginCapabilitiesRequest{}
		})

		JustBeforeEach(func() {
			expectedResponse, err = cs.GetPluginCapabilities(context, request)
		})

		It("returns the plugin capabilities", func() {
			Expect(expectedResponse).NotTo(BeNil())
			Expect(err).ToNot(HaveOccurred())
			Expect(expectedResponse.Capabilities).To(HaveLen(1))
			service := expectedResponse.Capabilities[0].GetService()
			Expect(service).NotTo(BeNil())
			Expect(service.GetType()).To(Equal(PluginCapability_Service_CONTROLLER_SERVICE))
		})
	})

	Describe("CreateSnapshot", func() {
		Context("when provided with a CreateSnapshotRequest", func() {
			It("returns an unimplemented error", func() {
				_, err = cs.CreateSnapshot(context, &CreateSnapshotRequest{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Snapshots not implemented"))
			})
		})
	})

	Describe("DeleteSnapshot", func() {
		Context("when provided with a DeleteSnapshotRequest", func() {
			It("returns an unimplemented error", func() {
				_, err = cs.DeleteSnapshot(context, &DeleteSnapshotRequest{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Snapshots not implemented"))
			})
		})
	})

	Describe("ListSnapshots", func() {
		var (
			request          *ListSnapshotsRequest
			expectedResponse *ListSnapshotsResponse
		)
		Context("when provided with a ListSnapshotsRequest", func() {
			BeforeEach(func() {
				request = &ListSnapshotsRequest{}
			})

			JustBeforeEach(func() {
				expectedResponse, err = cs.ListSnapshots(context, request)
			})

			It("returns an empty list of snapshots", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(*expectedResponse).To(Equal(ListSnapshotsResponse{}))
			})
		})
	})
})

type DummyContext struct{}

func (*DummyContext) Deadline() (deadline time.Time, ok bool) { return time.Time{}, false }

func (*DummyContext) Done() <-chan struct{} { return nil }

func (*DummyContext) Err() error { return nil }

func (*DummyContext) Value(key interface{}) interface{} { return nil }

func createSuccessful(ctx context.Context, cs ControllerServer, fakeOs *os_fake.FakeOs, volumeName string, vc []*VolumeCapability) *CreateVolumeResponse {
	createResponse, err := cs.CreateVolume(ctx, &CreateVolumeRequest{
		Name:               volumeName,
		VolumeCapabilities: vc,
	})
	Expect(err).To(BeNil())
	return createResponse
}

func deleteSuccessful(ctx context.Context, cs ControllerServer, volumeId string) *DeleteVolumeResponse {
	deleteResponse, err := cs.DeleteVolume(ctx, &DeleteVolumeRequest{
		VolumeId: volumeId,
	})
	Expect(err).To(BeNil())
	return deleteResponse
}
