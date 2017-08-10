package controller_test

import (
	"time"

	"code.cloudfoundry.org/goshims/filepathshim/filepath_fake"
	"code.cloudfoundry.org/goshims/osshim/os_fake"
	"github.com/jeffpak/local-controller-plugin/controller"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/types"
	. "github.com/paulcwarren/spec"
	"golang.org/x/net/context"
)

func VolumeIDMatcher(volID *VolumeID) GomegaMatcher {
	return WithTransform(func(entry *ListVolumesResponse_Result_Entry) *VolumeID {
		return entry.GetVolumeInfo().GetId()
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
		volID        *VolumeID
		vc           []*VolumeCapability
		volInfo      *VolumeInfo
		err          error
	)

	BeforeEach(func() {
		mountDir = "/path/to/mount"
		fakeOs = &os_fake.FakeOs{}
		fakeFilepath = &filepath_fake.FakeFilepath{}
		cs = controller.NewController(fakeOs, fakeFilepath, mountDir)
		context = &DummyContext{}
		volID = &VolumeID{Values: map[string]string{"volume_name": "vol-name"}}
		volumeName = "vol-name"
		vc = []*VolumeCapability{{Value: &VolumeCapability_Mount{Mount: &VolumeCapability_MountVolume{}}}}
		volInfo = &VolumeInfo{
			AccessMode: &AccessMode{Mode: AccessMode_UNKNOWN},
			Id:         volID}
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
				Reply: &CreateVolumeResponse_Result_{
					Result: &CreateVolumeResponse_Result{
						VolumeInfo: volInfo,
					},
				},
			}))
		})

		Context("when the Volume exists", func() {
			BeforeEach(func() {
				expectedResponse = createSuccessful(context, cs, fakeOs, volumeName, vc)
			})

			It("should succeed and respond with the existent volume", func() {
				Expect(*expectedResponse).To(Equal(CreateVolumeResponse{
					Reply: &CreateVolumeResponse_Result_{
						Result: &CreateVolumeResponse_Result{
							VolumeInfo: volInfo,
						},
					},
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
					Version:            &Version{},
					Name:               "",
					VolumeCapabilities: vc,
				}
			})
			JustBeforeEach(func() {
				createResponse, err = cs.CreateVolume(context, createVolReq)
			})
			It("should fail with an error response", func() {
				Expect(err).To(BeNil())
				Expect(createResponse.GetError()).NotTo(BeNil())
				Expect(createResponse.GetError().GetCreateVolumeError().GetErrorCode()).To(Equal(Error_CreateVolumeError_INVALID_VOLUME_NAME))
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
				Expect(err).To(BeNil())
				Expect(deleteVolResponse.GetError()).NotTo(BeNil())
				Expect(deleteVolResponse.GetError().GetDeleteVolumeError().GetErrorCode()).To(Equal(Error_DeleteVolumeError_INVALID_VOLUME_ID))
			})

			It("should fail if volume name is empty", func() {
				deleteVolResponse, err = cs.DeleteVolume(context, &DeleteVolumeRequest{
					VolumeId: &VolumeID{Values: map[string]string{"volume_name": ""}},
				})
				Expect(err).To(BeNil())
				Expect(deleteVolResponse.GetError()).NotTo(BeNil())
				Expect(deleteVolResponse.GetError().GetDeleteVolumeError().GetErrorCode()).To(Equal(Error_DeleteVolumeError_INVALID_VOLUME_ID))
			})

			It("should fail if no volume was found", func() {
				deleteVolResponse, err = cs.DeleteVolume(context, &DeleteVolumeRequest{
					VolumeId: &VolumeID{Values: map[string]string{"volume_name": "non-existent-volume"}},
				})
				Expect(err).To(BeNil())
				Expect(deleteVolResponse.GetError()).To(BeNil())
				Expect(deleteVolResponse.GetResult()).NotTo(BeNil())
			})

			Context("when the volume has been created", func() {
				var (
					createVolResponse *CreateVolumeResponse
				)
				BeforeEach(func() {
					createVolResponse = createSuccessful(context, cs, fakeOs, volumeName, vc)
				})

				It("should delete the volume", func() {
					response := deleteSuccessful(context, cs, volID)
					Expect(response).NotTo(BeNil())
					Expect(response.GetResult()).NotTo(BeNil())

					listReq = &ListVolumesRequest{
						Version:    &Version{Major: 0, Minor: 0, Patch: 1},
						MaxEntries: 100,
					}

					listResp, err = cs.ListVolumes(context, listReq)
					Expect(err).NotTo(HaveOccurred())
					Expect(listResp).NotTo(BeNil())
					volID := createVolResponse.GetResult().GetVolumeInfo().GetId()
					Expect(listResp.GetResult().GetEntries()).NotTo(ContainElement(VolumeIDMatcher(volID)))
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
						&Version{Major: 0, Minor: 0, Patch: 1},
						volID,
						&VolumeMetadata{Values: map[string]string{}},
						&NodeID{Values: map[string]string{}},
						false,
					}
				})
				JustBeforeEach(func() {
					expectedResponse, err = cs.ControllerPublishVolume(context, request)
				})
				It("should return a ControllerPublishVolumeResponse", func() {
					Expect(*expectedResponse).NotTo(BeNil())
					Expect(expectedResponse.GetResult().GetPublishVolumeInfo()).NotTo(BeNil())
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
						&Version{Major: 0, Minor: 0, Patch: 1},
						volID,
						&VolumeMetadata{Values: map[string]string{}},
						&NodeID{Values: map[string]string{}},
					}
				})
				JustBeforeEach(func() {
					expectedResponse, err = cs.ControllerUnpublishVolume(context, request)
				})
				It("should return a ControllerUnpublishVolumeResponse", func() {
					Expect(*expectedResponse).NotTo(BeNil())
					Expect(expectedResponse.GetResult()).NotTo(BeNil())
				})
				Context("when a volume is unpublished for a second time", func() {
					JustBeforeEach(func() {
						expectedResponse, err = cs.ControllerUnpublishVolume(context, request)
					})
					It("should return a ControllerUnpublishVolumeResponse", func() {
						Expect(*expectedResponse).NotTo(BeNil())
						Expect(expectedResponse.GetResult()).NotTo(BeNil())
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
						&Version{Major: 0, Minor: 0, Patch: 1},
						volInfo,
						[]*VolumeCapability{{Value: &VolumeCapability_Mount{
							Mount: &VolumeCapability_MountVolume{
								MountFlags: []string{""},
							}}}}}
				})
				JustBeforeEach(func() {
					expectedResponse, err = cs.ValidateVolumeCapabilities(context, request)
				})
				It("should return a ValidateVolumeResponse", func() {
					Expect(err).To(BeNil())
					Expect(expectedResponse).To(Equal(&ValidateVolumeCapabilitiesResponse{
						Reply: &ValidateVolumeCapabilitiesResponse_Result_{
							Result: &ValidateVolumeCapabilitiesResponse_Result{Supported: true},
						}}))
				})
			})

			Context("when called with unsupported FsType capabilities", func() {
				BeforeEach(func() {
					request = &ValidateVolumeCapabilitiesRequest{
						&Version{Major: 0, Minor: 0, Patch: 1},
						volInfo,
						[]*VolumeCapability{{Value: &VolumeCapability_Mount{
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
						Reply: &ValidateVolumeCapabilitiesResponse_Result_{
							Result: &ValidateVolumeCapabilitiesResponse_Result{
								Supported: false,
								Message:   "Specifying FsType is unsupported.",
							},
						}}))
				})
			})

			Context("when called with unsupported MountFlag capabilities", func() {
				BeforeEach(func() {
					request = &ValidateVolumeCapabilitiesRequest{
						&Version{Major: 0, Minor: 0, Patch: 1},
						volInfo,
						[]*VolumeCapability{{Value: &VolumeCapability_Mount{
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
						Reply: &ValidateVolumeCapabilitiesResponse_Result_{
							Result: &ValidateVolumeCapabilitiesResponse_Result{
								Supported: false,
								Message:   "Specifying mount flags is unsupported.",
							}},
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
					&Version{Major: 0, Minor: 0, Patch: 1},
					10,
					"starting-token",
				}
				expectedResponse, err = cs.ListVolumes(context, request)
			})

			It("should return a response listing that volume", func() {
				Expect(err).NotTo(HaveOccurred())
				Expect(expectedResponse).NotTo(BeNil())
				Expect(expectedResponse.GetResult().GetEntries()).To(ContainElement(VolumeIDMatcher(volID)))
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
						&Version{Major: 0, Minor: 0, Patch: 1},
					}
				})
				JustBeforeEach(func() {
					expectedResponse, err = cs.GetCapacity(context, request)
				})
				It("should return a GetCapacityResponse", func() {
					Expect(*expectedResponse).NotTo(BeNil())
					Expect(expectedResponse.GetResult()).NotTo(BeNil())
					Expect(expectedResponse.GetResult().GetTotalCapacity()).NotTo(BeNil())
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
					request = &ControllerGetCapabilitiesRequest{
						&Version{Major: 0, Minor: 0, Patch: 1},
					}
				})
				JustBeforeEach(func() {
					expectedResponse, err = cs.ControllerGetCapabilities(context, request)
				})

				It("should return a listing all capabilities", func() {
					Expect(expectedResponse).NotTo(BeNil())
					capabilities := expectedResponse.GetResult().GetCapabilities()
					Expect(capabilities).To(HaveLen(4))
					Expect(capabilities[0].GetRpc().GetType()).To(Equal(ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME))
					Expect(capabilities[1].GetRpc().GetType()).To(Equal(ControllerServiceCapability_RPC_PUBLISH_UNPUBLISH_VOLUME))
					Expect(capabilities[2].GetRpc().GetType()).To(Equal(ControllerServiceCapability_RPC_LIST_VOLUMES))
					Expect(capabilities[3].GetRpc().GetType()).To(Equal(ControllerServiceCapability_RPC_GET_CAPACITY))
				})
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
		Version:            &Version{},
		Name:               volumeName,
		VolumeCapabilities: vc,
	})
	Expect(err).To(BeNil())
	return createResponse
}

func deleteSuccessful(ctx context.Context, cs ControllerServer, volumeID *VolumeID) *DeleteVolumeResponse {
	deleteResponse, err := cs.DeleteVolume(ctx, &DeleteVolumeRequest{
		Version:  &Version{},
		VolumeId: volumeID,
	})
	Expect(err).To(BeNil())
	return deleteResponse
}
